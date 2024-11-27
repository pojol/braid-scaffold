package handlers

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/constant/fields"
	"braid-scaffold/middleware"
	"braid-scaffold/states/gameproto"
	"braid-scaffold/states/user"
	"context"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/router"
	"github.com/pojol/braid/router/msg"
)

type EntityType struct{}

func Send2Client(ctx core.ActorContext, gateid string, header *router.Header, pmsg proto.Message) {
	body, _ := proto.Marshal(pmsg)

	ctx.Send(router.Target{ID: gateid, Ev: events.ClientResponse},
		msg.NewBuilder(context.TODO()).WithResHeader(header).WithResBody(body).Build(),
	)
}

func MKGetUserInfo(ctx core.ActorContext) core.IChain {

	entity := ctx.GetValue(EntityType{}).(*user.EntityWrapper)

	unpackCfg := &middleware.MessageUnpackCfg[gameproto.UserInfoReq]{}

	return &actor.DefaultChain{
		Before: []actor.EventHandler{
			middleware.MessageUnpack(unpackCfg),
		},
		Handler: func(mw *msg.Wrapper) error {

			now := time.Now()

			entity.TimeInfo.LoginTime = now.Unix()
			entity.TimeInfo.SyncTime = now.Unix()

			Send2Client(ctx, msg.GetReqField[string](mw, fields.KeyGateID), mw.Req.Header, &gameproto.UserInfoRes{
				Bag:      entity.Bag,
				User:     entity.User,
				TimeInfo: entity.TimeInfo,
			})

			return nil
		},
	}
}
