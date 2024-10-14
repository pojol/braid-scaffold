package actors

import (
	"braid-scaffold/chains"
	"context"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/def"
)

type dynamicRegisterActor struct {
	*actor.Runtime
	loader core.IActorLoader
}

func NewDynamicRegisterActor(p core.IActorBuilder) core.IActor {
	return &dynamicRegisterActor{
		Runtime: &actor.Runtime{Id: p.GetID(), Ty: def.ActorDynamicRegister, Sys: p.GetSystem()},
		loader:  p.GetLoader(),
	}
}

func (a *dynamicRegisterActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)

	a.RegisterEvent(chains.DynamicRegister, chains.MakeDynamicRegister)
}
