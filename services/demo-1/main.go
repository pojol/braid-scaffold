package main

import (
	"braid-scaffold/actors"
	"braid-scaffold/template"
	"fmt"
	"os"

	"github.com/pojol/braid/3rd/redis"
	"github.com/pojol/braid/core"
	"github.com/pojol/braid/core/cluster/node"
	"github.com/pojol/braid/lib/log"
)

// This is a demonstration service. After users pull it, they can copy the code for use in their own services and then delete it.

func main() {
	slog, _ := log.NewServerLogger("demo-1")
	log.SetSLog(slog)
	defer log.Sync()

	// mock
	os.Setenv("NODE_ID", "demo-1")

	// mock redis
	redis.BuildClientWithOption(redis.WithAddr("redis://127.0.0.1:6379/0"))

	nodeCfg, actorTypes, err := template.ParseConfig("../../template/demo-1.yml", "../../template/actor_template.yml")
	if err != nil {
		panic(err)
	}

	factory := actors.BuildActorFactory(actorTypes)
	loader := actors.BuildDefaultActorLoader(factory)

	nod := node.BuildProcessWithOption(
		core.WithSystem(
			node.BuildSystemWithOption(nodeCfg.ID, loader),
		),
	)

	for _, base := range actorTypes {
		if base.Category == "core" {
			builder := nod.System().Loader(base.Name).WithID(nodeCfg.ID + "_" + base.Name)
			_, err = builder.Build()
			if err != nil {
				panic(err.Error())
			}
		}
	}

	for _, regActor := range nodeCfg.ActorOpts {
		builder := nod.System().Loader(regActor.Name).WithID(nodeCfg.ID + "_" + regActor.Name)
		for key, val := range regActor.Options {
			builder.WithOpt(key, val)
		}
		_, err = builder.Build()
		if err != nil {
			panic(err.Error())
		}
	}

	err = nod.Init()
	if err != nil {
		panic(fmt.Errorf("node init err %v", err.Error()))
	}

	nod.Update()

	fmt.Println("start http server succ")
	nod.WaitClose() // watch node exit signal
}
