package actors

import (
	"braid-scaffold/chains"
	"context"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/def"
)

type dynamicPickerActor struct {
	*actor.Runtime
}

func NewDynamicPickerActor(p core.IActorBuilder) core.IActor {
	return &dynamicPickerActor{
		Runtime: &actor.Runtime{Id: p.GetID(), Ty: def.ActorDynamicPicker, Sys: p.GetSystem()},
	}
}

func (a *dynamicPickerActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)
	a.RegisterEvent(chains.DynamicPick, chains.MakeDynamicPick)
}
