package session

import (
	"braid-scaffold/chains"
	"braid-scaffold/constant/fields"
	"braid-scaffold/states/gameproto"
	"braid-scaffold/template"
	"bytes"
	"context"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pojol/braid/def"
	"github.com/pojol/braid/lib/log"
	"github.com/pojol/braid/lib/token"
	"github.com/pojol/braid/lib/unbounded"
	"github.com/pojol/braid/router"
	"github.com/pojol/braid/router/msg"
)

type SendCallback func(target router.Target, mw *msg.Wrapper) error

var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

type Session struct {
	mgr  *Mgr
	conn *websocket.Conn
	uid  string
	sid  string

	callback SendCallback

	writeQueue *unbounded.Unbounded
	readQueue  *unbounded.Unbounded

	closeChan chan struct{} // Channel for close signal
	closeOnce sync.Once     // Ensure close only once

	wg         sync.WaitGroup // Wait for all goroutines to finish
	state      int32          // session state（atomic）
	createTime time.Time

	lastHeartbeat time.Time
}

const (
	StateConnected  = iota // Connected
	StateAuthorized        // Authorized
	StateClosed            // Closed

	HeartbeatInterval = 30 * time.Second
	HeartbeatTimeout  = 90 * time.Second
)

func NewSession(conn *websocket.Conn, handler SendCallback, m *Mgr) *Session {
	s := &Session{
		sid:        uuid.NewString(),
		mgr:        m,
		conn:       conn,
		readQueue:  unbounded.NewUnbounded(),
		writeQueue: unbounded.NewUnbounded(),
		closeChan:  make(chan struct{}),
		callback:   handler,
		state:      StateConnected,
		createTime: time.Now(),
	}

	s.wg.Add(3)

	go s.readLoop()
	go s.writeLoop()
	go s.heartbeatLoop()

	return s
}

func (s *Session) BindUID(uid string) {
	s.uid = uid
	atomic.StoreInt32(&s.state, StateAuthorized)
}

// EnqueueRead adds a message to the read queue => service
func (s *Session) EnqueueRead(mw *msg.Wrapper) {
	s.readQueue.Put(mw)
}

// EnqueueWrite adds a message to the write queue => client
func (s *Session) EnqueueWrite(mw *msg.Wrapper) {
	s.writeQueue.Put(mw)
}

func (s *Session) heartbeatLoop() {
	defer s.wg.Done()
	ticker := time.NewTicker(HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.closeChan:
			return
		case <-ticker.C:
			// 检查是否超时
			if time.Since(s.lastHeartbeat) > HeartbeatTimeout {
				log.WarnF("session %v heartbeat timeout, closing connection", s.sid)
				s.Close()
				return
			}

			// 发送心跳包
			heartbeat := msg.NewBuilder(context.TODO()).WithResHeader(&router.Header{
				Event: "heartbeat",
			}).Build()
			s.EnqueueWrite(heartbeat)
		}
	}
}

func (s *Session) readLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeChan:
			return
		case mw := <-s.readQueue.Get():
			realmsg := mw.(*msg.Wrapper)
			s.readQueue.Load()

			var actorid, actorty string

			if realmsg.Req.Header.Event == "" {
				log.DebugF("session %v read empty event message", s.sid)
				return
			}

			switch realmsg.Req.Header.Event {
			case chains.API_GuestLogin:
				actorid = def.SymbolLocalFirst
				actorty = template.ACTOR_LOGIN
			case chains.API_Heartbeat:
				s.lastHeartbeat = time.Now()
				// If users need to handle business logic through heartbeat, the message can be passed downstream
				return
			default:
				actorid = s.uid
				actorty = template.ACTOR_USER
			}

			realmsg.ToBuilder().WithReqCustomFields(fields.SessionID(s.sid))

			err := s.callback(router.Target{
				ID: actorid,
				Ty: actorty,
				Ev: realmsg.Req.Header.Event,
			}, realmsg)

			if err != nil {
				log.WarnF("session %v handle message %v err %v", s.sid, realmsg.Req.Header.Event, err.Error())
			}
		}
	}
}

func (s *Session) writeLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeChan:
			return
		case mw := <-s.writeQueue.Get():
			realmsg := mw.(*msg.Wrapper)
			s.writeQueue.Load()

			// Get a buffer from the pool
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset() // Clear the buffer for reuse

			resHeader := gameproto.MsgHeader{
				Event: realmsg.Res.Header.Event,
				Token: realmsg.Res.Header.Token,
			}

			if resHeader.Event == chains.API_GuestLogin {
				userID, err := token.Parse(resHeader.Token)
				if err != nil {
					log.WarnF("bind uid, but token %v parse err %v", resHeader.Token, err)
				} else {
					if existingSession := s.mgr.GetSessionByUID(userID); existingSession != nil && existingSession.sid != s.sid {
						// Close the existing session
						log.InfoF("kicking out existing session for uid %v", userID)
						existingSession.Close()
					}
					s.mgr.BindUID(s.sid, userID)
				}
			}

			resHeaderByt, _ := proto.Marshal(&resHeader)
			binary.Write(buf, binary.LittleEndian, uint16(len(resHeaderByt)))
			binary.Write(buf, binary.LittleEndian, resHeaderByt)
			binary.Write(buf, binary.LittleEndian, realmsg.Res.Body)

			err := s.conn.WriteMessage(websocket.BinaryMessage, buf.Bytes())
			if err != nil {
				log.WarnF("%v write messge 2 client err %v", s.uid, err.Error())
			}

			// Put the buffer back in the pool immediately after use
			bufferPool.Put(buf)
		}
	}
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		close(s.closeChan)
		s.conn.Close()
		s.wg.Wait()
	})
}
