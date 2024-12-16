package session

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/constant/fields"
	"braid-scaffold/states/gameproto"
	"braid-scaffold/template"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
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
	"github.com/pojol/braid/router/msg"
)

type SendCallback func(idOrSymbol, actorType, event string, mw *msg.Wrapper) error

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
	StateConnected     = iota // Connected
	StateAuthorized           // Authorized
	StateWaitReconnect        // WaitReconnect
	StateClosed               // Closed

	HeartbeatInterval = 5 * time.Second
	HeartbeatTimeout  = 20 * time.Second
	ReconnectTimeout  = 60 * time.Second
)

func NewSession(conn *websocket.Conn, handler SendCallback, m *Mgr) *Session {
	s := &Session{
		sid:           uuid.NewString(),
		mgr:           m,
		conn:          conn,
		readQueue:     unbounded.NewUnbounded(),
		writeQueue:    unbounded.NewUnbounded(),
		closeChan:     make(chan struct{}),
		callback:      handler,
		state:         StateConnected,
		createTime:    time.Now(),
		lastHeartbeat: time.Now(),
	}

	s.wg.Add(3) // 注意，这个数量是用于监听三个 goroutine 是否成功退出的

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
				log.InfoF("session %v heartbeat timeout", s.sid)
				if atomic.CompareAndSwapInt32(&s.state, StateAuthorized, StateWaitReconnect) {
					log.InfoF("session %v heartbeat timeout, waiting for reconnect", s.sid)

					// 启动重连超时检查
					go func() {
						select {
						case <-time.After(ReconnectTimeout):
							// 如果超过重连时间限制，才真正关闭会话
							if atomic.LoadInt32(&s.state) == StateWaitReconnect {
								log.InfoF("session %v reconnect timeout, closing connection", s.sid)
								s.Close()
							}
						case <-s.closeChan:
							return
						}
					}()
				}
				return
			}
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
				continue
			}

			switch realmsg.Req.Header.Event {
			case events.API_GuestLogin:
				actorid = def.SymbolLocalFirst
				actorty = template.ACTOR_LOGIN
			case events.API_Heartbeat:
				s.lastHeartbeat = time.Now()
				log.DebugF("receive heartbeat from session %v", s.sid)
				// If users need to handle business logic through heartbeat, the message can be passed downstream
				continue
			default:
				actorid = s.uid
				actorty = template.ACTOR_USER
			}

			realmsg.Ctx = context.WithValue(realmsg.Ctx, "request_id", uuid.NewString())

			realmsg.ToBuilder().WithReqCustomFields(fields.SessionID(s.sid))
			err := s.callback(actorid, actorty, realmsg.Req.Header.Event, realmsg)
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

			if resHeader.Event == events.API_GuestLogin /* or other login */ {
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

func (s *Session) Reconnect(newConn *websocket.Conn) error {
	if atomic.LoadInt32(&s.state) != StateWaitReconnect {
		return errors.New("session not in reconnect state")
	}

	s.conn = newConn
	s.lastHeartbeat = time.Now()
	atomic.StoreInt32(&s.state, StateAuthorized)

	log.InfoF("session %v successfully reconnected", s.sid)
	return nil
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		close(s.closeChan)

		if s.uid != "" {
			// notify user logout
		}

		err := s.mgr.RemoveSession(s.sid)
		if err != nil {
			log.WarnF("session disConnect err %v user %v session %v", err.Error(), s.uid, s.sid)
		}

		s.conn.Close()
		s.wg.Wait()
	})
}
