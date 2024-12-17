package actors

import (
	"braid-scaffold/constant/events"
	"braid-scaffold/handlers"
	"braid-scaffold/template"
	"context"

	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/actor"
)

type loginActor struct {
	*actor.Runtime
}

func NewLoginActor(p core.IActorBuilder) core.IActor {
	return &loginActor{
		Runtime: &actor.Runtime{Id: p.GetID(), Ty: template.ACTOR_LOGIN, Sys: p.GetSystem()},
	}
}

func (a *loginActor) Init(ctx context.Context) {
	a.Runtime.Init(ctx)

	a.OnEvent(events.API_GuestLogin, handlers.MkGuestLogin)
	// other login
}
