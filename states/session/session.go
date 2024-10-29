package session

import (
	"braid-scaffold/chains"
	"braid-scaffold/states/gameproto"
	"braid-scaffold/template"
	"bytes"
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
)

type SendCallback func(target router.Target, msg *router.MsgWrapper) error

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

	closeChan chan struct{} // 用于关闭的信号通道
	closeOnce sync.Once     // 确保只关闭一次

	wg         sync.WaitGroup // 等待所有 goroutine 结束
	state      int32          // 会话状态（atomic）
	createTime time.Time
}

const (
	StateConnected  = iota // 已连接
	StateAuthorized        // 已认证
	StateClosed            // 已关闭
)

func NewSession(conn *websocket.Conn, handler SendCallback, m *Mgr) *Session {
	s := &Session{
		sid:        uuid.NewString(),
		mgr:        m,
		conn:       conn,
		readQueue:  unbounded.NewUnbounded(),
		writeQueue: unbounded.NewUnbounded(),
		closeChan:  make(chan struct{}),
		state:      StateConnected,
		createTime: time.Now(),
	}

	s.wg.Add(2) // 为读写 goroutine 添加计数

	go s.readLoop()
	go s.writeLoop()

	return s
}

func (s *Session) BindUID(uid string) {
	s.uid = uid
	atomic.StoreInt32(&s.state, StateAuthorized)
}

// EnqueueRead 将消息加入读队列
func (s *Session) EnqueueRead(msg *router.MsgWrapper) {
	s.readQueue.Put(msg)
}

// EnqueueWrite 将消息加入写队列
func (s *Session) EnqueueWrite(msg *router.MsgWrapper) {
	s.writeQueue.Put(msg)
}

// readLoop 处理读取消息
func (s *Session) readLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeChan:
			return
		case msg := <-s.readQueue.Get():
			realmsg := msg.(*router.MsgWrapper)
			var actorid, actorty string

			switch realmsg.Req.Header.Event {
			case chains.API_GuestLogin:
				actorid = def.SymbolLocalFirst
				actorty = template.ACTOR_LOGIN
			default:
				userID, err := token.Parse(realmsg.Req.Header.Token)
				if err != nil {
					log.WarnF("recv user message parse token %v err %v", realmsg.Req.Header.Token, err.Error())
					s.readQueue.Load()
					return
				}
				actorid = userID
				actorty = template.ACTOR_USER
			}

			s.callback(router.Target{
				ID: actorid,
				Ty: actorty,
				Ev: realmsg.Req.Header.Event,
			}, realmsg)

			s.readQueue.Load()
		}
	}
}

// writeLoop 处理发送消息
func (s *Session) writeLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.closeChan:
			return
		case msg := <-s.writeQueue.Get():
			realmsg := msg.(*router.MsgWrapper)

			// Get a buffer from the pool
			buf := bufferPool.Get().(*bytes.Buffer)
			buf.Reset() // Clear the buffer for reuse

			resHeader := gameproto.MsgHeader{
				Event: realmsg.Req.Header.Event,
				Token: realmsg.Req.Header.Token,
			}

			if resHeader.Event == chains.API_GuestLogin { // 并且没有错误
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

			s.writeQueue.Load()
		}
	}
}

func (s *Session) Close() {
	s.closeOnce.Do(func() {
		close(s.closeChan)
		s.conn.Close()
		s.wg.Wait() // 等待所有 goroutine 结束
	})
}
