package chains

import (
	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/def"
	"github.com/pojol/braid/router"
)

func MakeDynamicPick(ctx core.ActorContext) core.IChain {
	return &actor.DefaultChain{

		Handler: func(mw *router.MsgWrapper) error {

			actor_ty := mw.Req.Header.Custom["actor_ty"]

			// Select a node with low weight and relatively fewer registered actors of this type
			nodeaddr, err := ctx.AddressBook().GetLowWeightNodeForActor(mw.Ctx, actor_ty)
			if err != nil {
				return err
			}

			// dispatcher to picker node
			return ctx.Call(router.Target{ID: nodeaddr.Node + "_" + "register", Ty: def.ActorDynamicRegister, Ev: def.EvDynamicRegister}, mw)
		},
	}
}

func MakeDynamicRegister(ctx core.ActorContext) core.IChain {
	return &actor.DefaultChain{

		Handler: func(mw *router.MsgWrapper) error {

			actor_ty := mw.Req.Header.Custom["actor_ty"]
			actor_id := mw.Req.Header.Custom["actor_id"]

			builder := ctx.Loader(actor_ty)
			builder.WithID(actor_id)

			for k, v := range mw.Req.Header.Custom {
				builder.WithOpt(k, v)
			}

			actor, err := builder.Build()
			if err != nil {
				return err
			}

			mw.Req.Header.PrevActorType = def.ActorDynamicRegister

			actor.Init(mw.Ctx)
			go actor.Update()

			return nil
		},
	}
}
