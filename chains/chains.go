package chains

// Built-in actor chains
const (
	// EvDynamicPick is used to pick an actor
	// customOptions:
	// - actor_id: string
	// - actor_ty: string
	DynamicPick = "braid.chains.dynamic_pick"

	// EvDynamicRegister is used to register an actor
	// customOptions:
	// - actor_ty: string
	DynamicRegister = "braid.chains.dynamic_register"

	// EvUnregister is used to unregister an actor
	// customOptions:
	// - actor_id: string
	UnregisterActor = "braid.chains.unregister_actor"

	// EvHttpHello is used to handle http requests
	HttpHello = "braid.chains.http_hello"
)
