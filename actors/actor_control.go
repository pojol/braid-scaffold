package actors

import (
	"braid-scaffold/chains"
	"context"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/def"
)

type controlActor struct {
	*actor.Runtime
}

func NewControlActor(p core.IActorBuilder) core.IActor {
	return &controlActor{
		Runtime: &actor.Runtime{Id: p.GetID(), Ty: def.ActorControl, Sys: p.GetSystem()},
	}
}

func (a *controlActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)

	a.RegisterEvent(chains.UnregisterActor, chains.MakeUnregisterActor)
}
