package handlers

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/constant/fields"
	"braid-scaffold/errcode"
	"braid-scaffold/middleware"
	"braid-scaffold/states/gameproto"
	"braid-scaffold/states/user"
	"braid-scaffold/template"
	"context"
	"time"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/lib/dismutex"
	"github.com/pojol/braid/lib/log"
	"github.com/pojol/braid/lib/token"
	"github.com/pojol/braid/router/msg"
)

type SessionState struct{}

func MkGuestLogin(ctx core.ActorContext) core.IChain {

	unpackCfg := &middleware.MessageUnpackCfg[gameproto.GuestLoginReq]{}

	return &actor.DefaultChain{
		Before: []actor.EventHandler{middleware.MessageUnpack(unpackCfg)},
		Handler: func(mw *msg.Wrapper) error {
			req := unpackCfg.Msg.(*gameproto.GuestLoginReq)

			if req.Acc == "" {
				return errcode.NameLegalErr()
			}

			return loginImpl(ctx, mw, "guest", req.Acc)
		},
	}
}

func loginImpl(ctx core.ActorContext, mw *msg.Wrapper, loginTy string, id string) error {
	muid, err := dismutex.Lock(mw.Ctx, id)
	if err != nil {
		return err
	}

	info, err := ctx.AddressBook().GetByID(mw.Ctx, id)

	if err == nil && info.ActorId == id {
		msgbuild := mw.ToBuilder()
		msgbuild.WithReqCustomFields(fields.ActorID(id))
		msgbuild.WithReqCustomFields(fields.MutexID(muid))
		err = ctx.Call(id, template.ACTOR_USER, events.Ev_UserRefreshSession, msgbuild.Build())
		if err != nil {
			log.WarnF("login user %v refresh session err %v", id, err.Error())
			goto ext
		}
	} else {
		entity := user.NewEntityWapper(id)

		if entity.IsExist() {
			newToken, err := token.Create(entity.ID)
			if err != nil {
				goto ext
			}

			entity.User.Token = newToken
			log.InfoF("user %v refresh token %v", entity.ID, newToken)
		} else {
			entity.ID = id
			entity.User.Token, _ = token.Create(entity.ID)
			entity.TimeInfo.CreateTime = time.Now().Unix()

			err := entity.Sync(context.TODO(), true)
			if err != nil {
				goto ext
			}

			log.InfoF("login user %v create succ", entity.ID)
		}

		err = ctx.Loader(template.ACTOR_USER).WithID(entity.ID).
			WithOpt(fields.KeyGateID, msg.GetReqCustomField[string](mw, fields.KeyGateID)).
			WithOpt(fields.KeySessionID, msg.GetReqCustomField[string](mw, fields.KeySessionID)).
			WithOpt(fields.KeyMutexID, muid).
			Picker(mw.Ctx)
		if err != nil {
			goto ext
		}
	}

ext:
	if err != nil {
		dismutex.Unlock(mw.Ctx, id, muid)
	}

	return nil
}
