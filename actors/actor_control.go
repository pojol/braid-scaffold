package actors

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/handlers"
	"braid-scaffold/template"
	"context"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
)

// controlActor manages actor lifecycle operations within a node.
// It handles:
//   - Actor exit operations
//   - Actor reentry operations
//   - One controller per node
type controlActor struct {
	*actor.Runtime
}

func NewControlActor(p core.IActorBuilder) core.IActor {
	return &controlActor{
		Runtime: &actor.Runtime{Id: p.GetID(), Ty: template.ACTOR_CONTROL, Sys: p.GetSystem()},
	}
}

func (a *controlActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)

	a.OnEvent(events.UnregisterActor, handlers.MakeUnregisterActor)
}
