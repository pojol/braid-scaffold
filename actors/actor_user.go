package actors

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/constant/fields"
	"braid-scaffold/handlers"
	"braid-scaffold/states/gameproto"
	"braid-scaffold/states/user"
	"braid-scaffold/template"
	"context"
	"fmt"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/def"
	"github.com/pojol/braid/lib/log"
	"github.com/pojol/braid/router"
	"github.com/pojol/braid/router/msg"
)

type UserActor struct {
	*actor.Runtime
	gateID    string
	sessionID string
	entity    *user.EntityWrapper
}

const ActorExpirationSeconds = 60 * 1000 * 10 // 10 min

func NewUserActor(p core.IActorBuilder) core.IActor {
	return &UserActor{
		Runtime:   &actor.Runtime{Id: p.GetID(), Ty: p.GetType(), Sys: p.GetSystem()},
		gateID:    p.GetOpt(fields.KeyGateID),
		sessionID: p.GetOpt(fields.KeySessionID),
		entity:    user.NewEntityWapper(p.GetID()),
	}
}

func (a *UserActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)
	err := a.entity.Load(context.TODO())
	if err != nil {
		panic(fmt.Errorf("load user actor err %v", err.Error()))
	}

	a.Context().WithValue(handlers.EntityType{}, a.entity)

	a.RegisterEvent(events.API_GetUserInfo, handlers.MKGetUserInfo)
	// more events ...

	a.RegisterTimer(1000, 60*1000, func(i interface{}) error {

		// check zombie
		if a.entity.TimeInfo.SyncTime+ActorExpirationSeconds < time.Now().Unix() {

			msgbuild := msg.NewBuilder(context.TODO())
			msgbuild.WithReqCustomFields(fields.ActorID(a.Id))
			msgbuild.WithReqCustomFields(fields.ActorTy(a.Ty))
			a.Sys.Send(def.SymbolLocalFirst, template.ACTOR_CONTROL, events.UnregisterActor, msgbuild.Build())
		}

		return nil
	}, nil)

	a.loginCallback() // 完成 actor 具柄注册后通知给客户端，不然消息可能会在消息具柄注册前过来
	log.InfoF("user actor %v init succ", a.entity.ID)
}

func (a *UserActor) loginCallback() {
	body, _ := proto.Marshal(&gameproto.GuestLoginRes{
		Acc:   a.entity.ID,
		Token: a.entity.User.Token,
	})

	a.Sys.Send(a.gateID, "", events.ClientResponse,
		msg.NewBuilder(context.TODO()).WithResHeader(&router.Header{
			Event: events.API_GuestLogin, // tmp
			Token: a.entity.User.Token,
		}).WithResCustomFields(fields.SessionID(a.sessionID)).WithResBody(body).Build(),
	)
}
