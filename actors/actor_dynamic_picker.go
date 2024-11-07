package actors

import (
	"braid-scaffold/chains"
	"braid-scaffold/template"
	"context"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
)

// dynamicPickerActor selects nodes for dynamically constructed actors.
// It:
//   - Chooses nodes with lower load in the cluster
//   - Forwards registration messages to the selected node's registrar
//   - Helps balance actor distribution across the cluster
type dynamicPickerActor struct {
	*actor.Runtime
}

func NewDynamicPickerActor(p core.IActorBuilder) core.IActor {
	return &dynamicPickerActor{
		Runtime: &actor.Runtime{Id: p.GetID(), Ty: template.ACTOR_DYNAMIC_PICKER, Sys: p.GetSystem()},
	}
}

func (a *dynamicPickerActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)
	a.RegisterEvent(chains.DynamicPick, chains.MakeDynamicPick)
}
