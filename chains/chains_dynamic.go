package chains

import (
	"braid-scaffold/constant"
	"braid-scaffold/template"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/router"
)

func MakeDynamicPick(ctx core.ActorContext) core.IChain {
	return &actor.DefaultChain{

		Handler: func(mw *router.MsgWrapper) error {

			actor_ty := mw.GetReqCustomStr(constant.CustomActorType)

			// Select a node with low weight and relatively fewer registered actors of this type
			nodeaddr, err := ctx.AddressBook().GetLowWeightNodeForActor(mw.Ctx, actor_ty)
			if err != nil {
				return err
			}

			// dispatcher to picker node
			return ctx.Call(router.Target{ID: nodeaddr.Node + "_" + template.ACTOR_DYNAMIC_REGISTER, Ty: template.ACTOR_DYNAMIC_REGISTER, Ev: DynamicRegister}, mw)
		},
	}
}

func MakeDynamicRegister(ctx core.ActorContext) core.IChain {
	return &actor.DefaultChain{

		Handler: func(mw *router.MsgWrapper) error {

			actor_ty := mw.GetReqCustomStr(constant.CustomActorType)
			actor_id := mw.GetReqCustomStr(constant.CustomActorID)

			builder := ctx.Loader(actor_ty)
			builder.WithID(actor_id)

			for k, v := range mw.Req.Header.Custom {
				builder.WithOpt(k, v)
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
