package handlers

import (
	"braid-scaffold/constant/fields"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/lib/log"
	"github.com/pojol/braid/router/msg"
)

func MakeUnregisterActor(ctx core.ActorContext) core.IChain {
	return &actor.DefaultChain{
		Handler: func(mw *msg.Wrapper) error {

			actor_id := msg.GetReqCustomField[string](mw, fields.KeyActorID)
			actor_ty := msg.GetReqCustomField[string](mw, fields.KeyActorTy)

			err := ctx.Unregister(actor_id, actor_ty)
			if err != nil {
				log.WarnF("[braid.actor_control] unregister actor %v err %v", actor_id, err)
			}

			return nil
		},
	}
}
