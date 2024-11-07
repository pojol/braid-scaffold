package actors

import (
	"braid-scaffold/chains"
	"braid-scaffold/template"
	"context"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
)

// dynamicRegisterActor handles actor registration within a node.
// It:
//   - Listens for actor registration messages
//   - Registers actors to the current node
//   - Exists as a singleton instance on each node
type dynamicRegisterActor struct {
	*actor.Runtime
	loader core.IActorLoader
}

func NewDynamicRegisterActor(p core.IActorBuilder) core.IActor {
	return &dynamicRegisterActor{
		Runtime: &actor.Runtime{Id: p.GetID(), Ty: template.ACTOR_DYNAMIC_REGISTER, Sys: p.GetSystem()},
		loader:  p.GetLoader(),
	}
}

func (a *dynamicRegisterActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)

	a.RegisterEvent(chains.DynamicRegister, chains.MakeDynamicRegister)
}
