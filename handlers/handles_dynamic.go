package handlers

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/constant/fields"
	"braid-scaffold/template"
	"fmt"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/router"
	"github.com/pojol/braid/router/msg"
)

func MakeDynamicPick(ctx core.ActorContext) core.IChain {
	return &actor.DefaultChain{

		Handler: func(mw *msg.Wrapper) error {

			actor_ty := msg.GetReqField[string](mw, fields.KeyActorTy)

			// Select a node with low weight and relatively fewer registered actors of this type
			nodeaddr, err := ctx.AddressBook().GetLowWeightNodeForActor(mw.Ctx, actor_ty)
			if err != nil {
				return err
			}

			// dispatcher to picker node
			return ctx.Call(router.Target{ID: nodeaddr.Node + "_" + template.ACTOR_DYNAMIC_REGISTER, Ty: template.ACTOR_DYNAMIC_REGISTER, Ev: events.DynamicRegister}, mw)
		},
	}
}

func MakeDynamicRegister(ctx core.ActorContext) core.IChain {
	return &actor.DefaultChain{

		Handler: func(mw *msg.Wrapper) error {

			actor_ty := msg.GetReqField[string](mw, fields.KeyActorTy)
			actor_id := msg.GetReqField[string](mw, fields.KeyActorID)

			builder := ctx.Loader(actor_ty)
			builder.WithID(actor_id)

			custom, err := mw.GetReqCustomMap()
			if err != nil {
				return fmt.Errorf("dynamic register get custom map err %w", err)
			}

			for k, v := range custom {
				builder.WithOpt(k, v.(string))
			}

			actor, err := builder.Register()
			if err != nil {
				return err
			}

			mw.Req.Header.PrevActorType = template.ACTOR_DYNAMIC_REGISTER

			actor.Init(mw.Ctx)
			go actor.Update()

			return nil
		},
	}
}
