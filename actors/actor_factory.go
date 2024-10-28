package actors

import (
	"braid-scaffold/template"

	"github.com/pojol/braid/core"
)

// MockActorFactory is a factory for creating actors
type MockActorFactory struct {
	actors       []template.RegisteredActorConfig
	constructors map[string]*core.ActorConstructor
}

// NewActorFactory create new actor factory
func BuildActorFactory(actorcfg []template.RegisteredActorConfig) *MockActorFactory {
	factory := &MockActorFactory{
		actors:       actorcfg,
		constructors: make(map[string]*core.ActorConstructor),
	}

	for _, v := range actorcfg {
		var create core.CreateFunc

		switch v.Name {
		case template.ACTOR_DYNAMIC_PICKER:
			create = NewDynamicPickerActor
		case template.ACTOR_DYNAMIC_REGISTER:
			create = NewDynamicRegisterActor
		case template.ACTOR_HTTP_ACCEPTOR:
			create = NewHttpAcceptorActor
		case template.ACTOR_CONTROL:
			create = NewControlActor
			// todo ...
		}

		factory.constructors[v.Name] = &core.ActorConstructor{
			Constructor:         create,
			ID:                  v.ID,
			Name:                v.Name,
			Weight:              v.Weight,
			NodeUnique:          v.Unique,
			GlobalQuantityLimit: v.Limit,
			Dynamic:             v.Dynamic,
		}

		if len(v.Options) > 0 {
			factory.constructors[v.Name].Options = v.Options
		} else {
			factory.constructors[v.Name].Options = make(map[string]string)
		}
	}

	return factory
}

func (factory *MockActorFactory) Get(actorType string) *core.ActorConstructor {
	if _, ok := factory.constructors[actorType]; ok {
		return factory.constructors[actorType]
	}

	return nil
}

func (factory *MockActorFactory) GetActors() []*core.ActorConstructor {
	actors := []*core.ActorConstructor{}
	for _, v := range factory.constructors {
		actors = append(actors, v)
	}
	return actors
}
