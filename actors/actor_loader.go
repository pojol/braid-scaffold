package actors

import (
	"braid-scaffold/chains"
	"braid-scaffold/constant"
	"braid-scaffold/template"
	"context"
	"fmt"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
	"github.com/pojol/braid/def"
	"github.com/pojol/braid/lib/log"
	"github.com/pojol/braid/router"
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

func (al *DefaultActorLoader) Pick(builder core.IActorBuilder) error {

	customOptions := make(map[string]string)

	for key, value := range builder.GetOptions() {
		customOptions[key] = fmt.Sprint(value)
	}

	customOptions[constant.CustomActorID] = builder.GetID()
	customOptions[constant.CustomActorType] = builder.GetType()

	go func() {
		err := builder.GetSystem().Call(router.Target{
			ID: def.SymbolWildcard,
			Ty: template.ACTOR_DYNAMIC_PICKER,
			Ev: chains.DynamicPick},
			router.NewMsgWrap(context.TODO()).WithReqHeader(&router.Header{
				Custom: customOptions,
			}).Build(),
		)
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

		_, err := builder.Register()
		if err != nil {
			log.InfoF("assign to node build actor %s err %v", actor.Name, err)
		}
	}
}
