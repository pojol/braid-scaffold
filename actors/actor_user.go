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
	"github.com/pojol/braid/lib/dismutex"
	"github.com/pojol/braid/lib/log"
	"github.com/pojol/braid/router"
	"github.com/pojol/braid/router/msg"
)

type UserActor struct {
	*actor.Runtime
	gateID    string
	sessionID string
	tmpmuid   string //
	offline   bool
	entity    *user.EntityWrapper
}

const ActorExpirationSeconds = 60 * 1000 * 10 // 10 min

func NewUserActor(p core.IActorBuilder) core.IActor {
	return &UserActor{
		Runtime:   &actor.Runtime{Id: p.GetID(), Ty: p.GetType(), Sys: p.GetSystem()},
		gateID:    p.GetOpt(fields.KeyGateID),
		sessionID: p.GetOpt(fields.KeySessionID),
		tmpmuid:   p.GetOpt(fields.KeyMutexID),
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

	a.OnEvent(events.API_GetUserInfo, handlers.MKGetUserInfo)
	a.OnEvent(events.Ev_UserRefreshSession, func(ctx core.ActorContext) core.IChain {
		return &actor.DefaultChain{
			Handler: func(w *msg.Wrapper) error {
				a.offline = false // active

				a.gateID = msg.GetReqCustomField[string](w, fields.KeyGateID)
				a.sessionID = msg.GetReqCustomField[string](w, fields.KeySessionID)
				a.tmpmuid = msg.GetReqCustomField[string](w, fields.KeyMutexID)

				a.loginCallback(w.Ctx)
				return nil
			},
		}
	})

	a.OnTimer(1000, 60*1000, func(i interface{}) error {
		if a.entity.TimeInfo.SyncTime+ActorExpirationSeconds < time.Now().Unix() && a.offline {

			msgbuild := msg.NewBuilder(context.TODO())
			msgbuild.WithReqCustomFields(fields.ActorID(a.Id))
			msgbuild.WithReqCustomFields(fields.ActorTy(a.Ty))

			a.Sys.Send(def.SymbolLocalFirst, template.ACTOR_CONTROL, events.UnregisterActor, msgbuild.Build())
		}

		return nil
	}, nil)

	a.loginCallback(ctx)
	log.InfoF("user actor %v init succ", a.entity.ID)
}

func (a *UserActor) loginCallback(ctx context.Context) {
	ok := dismutex.Unlock(context.TODO(), a.Id, a.tmpmuid)
	if !ok {
		log.WarnF("user actor %v distributed lock %v release failed", a.Id, a.tmpmuid)
	}

	body, _ := proto.Marshal(&gameproto.GuestLoginRes{
		Acc:   a.entity.ID,
		Token: a.entity.User.Token,
	})

	a.Sys.Send(a.gateID, "", events.ClientResponse,
		msg.NewBuilder(ctx).WithResHeader(&router.Header{
			Event: events.API_GuestLogin, // tmp
			Token: a.entity.User.Token,
		}).WithResCustomFields(fields.SessionID(a.sessionID)).WithResBody(body).Build(),
	)
}
