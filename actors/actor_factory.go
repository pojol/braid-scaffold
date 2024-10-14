package actors

import (
	"braid-scaffold/template"

	"github.com/pojol/braid/core"
)

// MockActorFactory is a factory for creating actors
type MockActorFactory struct {
	constructors map[string]*core.ActorConstructor
}

// NewActorFactory create new actor factory
func BuildActorFactory(actorcfg []template.ActorConfig) *MockActorFactory {
	factory := &MockActorFactory{
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

		factory.bind(v.Name, v.Unique, create)
	}

	return factory
}

// Bind associates an actor type with its constructor function
func (factory *MockActorFactory) bind(actorType string, unique bool, f core.CreateFunc) {
	factory.constructors[actorType] = &core.ActorConstructor{
		NodeUnique:  unique,
		Constructor: f,
	}
}

func (factory *MockActorFactory) Get(actorType string) *core.ActorConstructor {
	if _, ok := factory.constructors[actorType]; ok {
		return factory.constructors[actorType]
	}

	return nil
}
