package actors

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/constant/fields"
	"braid-scaffold/template"
	"context"
	"fmt"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/def"
	"github.com/pojol/braid/lib/log"
	"github.com/pojol/braid/router/msg"
)

// DefaultActorLoader manages actor loading and initialization in nodes.
// It:
//   - Loads non-dynamic actors from factory during node initialization
//   - Handles actor picking operations across the cluster
//   - Automatically locates and utilizes picker actors for selection
type DefaultActorLoader struct {
	factory core.IActorFactory
}

func BuildDefaultActorLoader(factory core.IActorFactory) core.IActorLoader {
	return &DefaultActorLoader{factory: factory}
}

func (al *DefaultActorLoader) Pick(ctx context.Context, builder core.IActorBuilder) error {

	msgbu := msg.NewBuilder(ctx)

	for key, value := range builder.GetOptions() {
		msgbu.WithReqCustomFields(msg.Attr{Key: key, Value: fmt.Sprint(value)})
	}

	msgbu.WithReqCustomFields(fields.ActorID(builder.GetID()))
	msgbu.WithReqCustomFields(fields.ActorTy(builder.GetType()))

	go func() {
		err := builder.GetSystem().Call(def.SymbolWildcard, template.ACTOR_DYNAMIC_PICKER, events.DynamicPick, msgbu.Build())
		if err != nil {
			log.WarnF("[braid.actorLoader] call dynamic picker err %v", err.Error())
		}
	}()

	return nil
}

// Builder selects an actor from the factory and provides a builder
func (al *DefaultActorLoader) Builder(ty string, sys core.ISystem) core.IActorBuilder {
	ac := al.factory.Get(ty)
	if ac == nil {
		return nil
	}

	builder := &actor.ActorLoaderBuilder{
		ISystem:          sys,
		ActorConstructor: *ac,
		IActorLoader:     al,
	}

	return builder
}

func (al *DefaultActorLoader) AssignToNode(node core.INode) {
	actors := al.factory.GetActors()

	for _, actor := range actors {

		if actor.Dynamic {
			continue
		}

		builder := al.Builder(actor.Name, node.System())
		if actor.ID == "" {
			actor.ID = actor.Name
		}

		builder.WithID(node.ID() + "_" + actor.ID)

		_, err := builder.Register(context.TODO())
		if err != nil {
			log.InfoF("assign to node build actor %s err %v", actor.Name, err)
		}
	}
}
