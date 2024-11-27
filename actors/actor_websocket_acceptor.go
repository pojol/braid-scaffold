package actors

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/constant/fields"
	"braid-scaffold/states/gameproto"
	"braid-scaffold/states/session"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/lib/log"
	"github.com/pojol/braid/lib/token"
	"github.com/pojol/braid/router"
	"github.com/pojol/braid/router/msg"
)

type websocketAcceptorActor struct {
	*actor.Runtime
	echoptr *echo.Echo
	Port    string
	KeyPath string
	PemPath string

	sessionMgr *session.Mgr
}

var (
	ErrInvalidMessageFormat = errors.New("invalid message format")
	ErrInvalidToken         = errors.New("invalid token")
	ErrSessionNotFound      = errors.New("session not found")
)

var (
	upgrader = websocket.Upgrader{}
)

func NewWSAcceptorActor(p core.IActorBuilder) core.IActor {

	echoptr := echo.New()
	echoptr.HideBanner = true

	return &websocketAcceptorActor{
		Runtime:    &actor.Runtime{Id: p.GetID(), Ty: p.GetType(), Sys: p.GetSystem()},
		echoptr:    echoptr,
		Port:       p.GetOpt("port"),
		KeyPath:    p.GetOpt("key_path"),
		PemPath:    p.GetOpt("pem_path"),
		sessionMgr: session.NewSessionMgr(),
	}
}

func (a *websocketAcceptorActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)

	recovercfg := middleware.DefaultRecoverConfig
	recovercfg.LogErrorFunc = func(c echo.Context, err error, stack []byte) error {
		log.ErrorF("recover err %v stack %v", err.Error(), string(stack))
		return nil
	}
	a.echoptr.Use(middleware.RecoverWithConfig(recovercfg))
	a.echoptr.Use(middleware.CORS())

	a.echoptr.GET("/ws", a.received)

	a.RegisterEvent(events.ClientResponse, func(ctx core.ActorContext) core.IChain {
		return &actor.DefaultChain{
			Handler: func(mw *msg.Wrapper) error {

				sid := msg.GetResField[string](mw, fields.KeySessionID)
				session := a.sessionMgr.GetSessionByID(sid)
				if session != nil {
					session.EnqueueWrite(mw)
				}

				return nil
			},
		}
	})
	a.RegisterEvent(events.ClientBroadcast, func(ctx core.ActorContext) core.IChain {
		return &actor.DefaultChain{
			Handler: func(mw *msg.Wrapper) error {
				uid := msg.GetResField[string](mw, fields.KeyUserID)
				session := a.sessionMgr.GetSessionByUID(uid)
				if session != nil {
					session.EnqueueWrite(mw)
				}
				return nil
			},
		}
	})

	a.RegisterTimer(1000, 1000*60*10, func(i interface{}) error {
		a.sessionMgr.CleanupExpiredSessions(time.Minute * 10)
		return nil
	}, nil)

	go func() {
		var err error
		log.InfoF("Starting WebSocket server on port %s", a.Port)
		if a.KeyPath != "" {
			err = a.echoptr.StartTLS(":"+a.Port, a.PemPath, a.KeyPath)
		} else {
			err = a.echoptr.Start(":" + a.Port)
		}

		if err != nil {
			log.InfoF("Failed to start echo server: %v", err.Error())
		}
	}()

	log.InfoF("init websocket actor succ")
}

func (a *websocketAcceptorActor) received(c echo.Context) error {

	ws, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	for {
		// Read
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.DebugF("read msg err %v", err.Error())
			break
		}

		header, err := parseMessageHeader(msg)
		if err != nil {
			log.WarnF("parse message header error: %v", err)
			continue
		}

		if err := a.handleMessage(context.Background(), header, msg, ws); err != nil {
			log.WarnF("handle message error: %v", err)
		}

	}

	return nil
}

func (a *websocketAcceptorActor) handleMessage(ctx context.Context, header *gameproto.MsgHeader, msgbody []byte, ws *websocket.Conn) error {

	var userID string
	var err error

	sendMsg := msg.NewBuilder(ctx).
		WithReqHeader(&router.Header{
			Token: header.Token,
			Event: header.Event,
		}).
		WithReqCustomFields(fields.GateID(a.Id)).
		WithReqBody(msgbody[2+binary.LittleEndian.Uint16(msgbody[:2]):]).
		Build()

	if header.Event == events.API_GuestLogin {

		session := a.sessionMgr.NewSession(ws, func(idOrSymbol, actorType, event string, mw *msg.Wrapper) error {
			return a.Sys.Send(idOrSymbol, actorType, event, mw)
		})

		session.EnqueueRead(sendMsg)

	} else {
		userID, err = token.Parse(header.Token)
		if err != nil {
			return ErrInvalidToken
		}

		session := a.sessionMgr.GetSessionByUID(userID)
		if session != nil {
			session.EnqueueRead(sendMsg)
		}
	}

	return nil
}

// Message Frame Format:
// ┌──────────────┬─────────────┬─────────────┐
// │ Header Size  │ Header Body │ Message Body│
// │   (2 bytes)  │    (var)    │    (var)    │
// └──────────────┴─────────────┴─────────────┘
func parseMessageHeader(msg []byte) (*gameproto.MsgHeader, error) {
	if len(msg) < 2 {
		return nil, ErrInvalidMessageFormat
	}

	headerLen := binary.LittleEndian.Uint16(msg[:2])
	if len(msg) < int(2+headerLen) {
		return nil, ErrInvalidMessageFormat
	}

	header := &gameproto.MsgHeader{}
	if err := proto.Unmarshal(msg[2:2+headerLen], header); err != nil {
		return nil, fmt.Errorf("unmarshal header error: %w", err)
	}

	return header, nil
}

func (a *websocketAcceptorActor) Exit() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.echoptr.Shutdown(ctx); err != nil {
		log.ErrorF("failed to shutdown server: %v", err)
	}

	a.sessionMgr.CloseAll()

	a.Runtime.Exit()
}
