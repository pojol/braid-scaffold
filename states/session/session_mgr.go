package session

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Mgr struct {
	sync.RWMutex
	sessions    map[string]*Session // sessionID -> Session
	uidSessions map[string]string
}

func NewSessionMgr() *Mgr {
	return &Mgr{
		sessions:    make(map[string]*Session),
		uidSessions: make(map[string]string),
	}
}

func (m *Mgr) NewSession(conn *websocket.Conn, sendcallback SendCallback) *Session {
	m.Lock()
	defer m.Unlock()

	// Add the connection to the session map
	newSession := NewSession(conn, sendcallback, m)
	m.sessions[newSession.sid] = newSession

	return newSession
}

func (m *Mgr) SessionCount() int {
	cnt := 0
	m.RLock()
	cnt = len(m.sessions)
	m.RUnlock()
	return cnt
}

func (m *Mgr) BindUID(sessionID, uid string) {
	m.Lock()
	defer m.Unlock()
	if session, ok := m.sessions[sessionID]; ok {
		m.uidSessions[uid] = sessionID
		session.BindUID(uid)
	}
}

func (m *Mgr) GetSessionByID(id string) *Session {
	m.RLock()
	defer m.RUnlock()

	return m.sessions[id]
}

func (m *Mgr) GetSessionByUID(uid string) *Session {
	m.RLock()
	defer m.RUnlock()
	if sessionID, ok := m.uidSessions[uid]; ok {
		if session, ok := m.sessions[sessionID]; ok {
			return session
		}
	}
	return nil
}

func (m *Mgr) CloseAll() {
	m.Lock()
	defer m.Unlock()

	// 关闭所有会话
	for _, session := range m.sessions {
		session.Close()
	}

	// 清理所有映射
	m.sessions = make(map[string]*Session)
	m.uidSessions = make(map[string]string)
}

func (m *Mgr) RemoveSession(sid string) error {
	// 获取 session 对象
	session, exists := m.sessions[sid]
	if !exists {
		return fmt.Errorf("session inconsistency for sid: %s", sid)
	}

	// 清理映射关系
	delete(m.sessions, sid)
	if session.uid != "" {
		delete(m.uidSessions, session.uid)
	}

	return nil
}

func (m *Mgr) CleanupExpiredSessions(timeout time.Duration) {
	m.Lock()
	defer m.Unlock()

	now := time.Now()
	for id, session := range m.sessions {
		// 清理超时的未认证会话
		if atomic.LoadInt32(&session.state) == StateConnected &&
			now.Sub(session.createTime) > timeout {
			session.Close()
			delete(m.sessions, id)
			// 如果有绑定的UID，也要清理
			if session.uid != "" {
				delete(m.uidSessions, session.uid)
			}
		}
	}
}
