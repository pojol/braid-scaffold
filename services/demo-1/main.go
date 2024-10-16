package main

import (
	"braid-scaffold/actors"
	"braid-scaffold/template"
	"fmt"
	"os"
	"strconv"

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
	os.Setenv("BRAID_NODE_ID", "demo-1")
	os.Setenv("BRAID_NODE_WEIGHT", "100")
	os.Setenv("BRAID_NODE_PORT", "22222")

	// mock redis
	redis.BuildClientWithOption(redis.WithAddr("redis://127.0.0.1:6379/0"))

	nodeCfg, err := template.ParseConfig("../../template/demo-1.yml", "../../template/actor_template.yml")
	if err != nil {
		panic(err)
	}

	realNodePort, err := strconv.Atoi(nodeCfg.Port)
	if err != nil {
		panic(fmt.Errorf("node port %s is not a valid integer", nodeCfg.Port))
	}

	realNodeWeight, err := strconv.Atoi(nodeCfg.Weight)
	if err != nil {
		panic(fmt.Errorf("node weight %s is not a valid integer", nodeCfg.Weight))
	}

	factory := actors.BuildActorFactory(nodeCfg.Actors)
	loader := actors.BuildDefaultActorLoader(factory)

	nod := node.BuildProcessWithOption(
		core.WithNodeID(nodeCfg.ID),
		core.WithWeight(realNodeWeight),
		core.WithLoader(loader),
		core.WithSystem(
			node.BuildSystemWithOption(nodeCfg.ID, loader, node.SystemWithAcceptor(realNodePort)),
		),
	)

	err = nod.Init()
	if err != nil {
		panic(fmt.Errorf("node init err %v", err.Error()))
	}

	nod.Update()

	fmt.Println("start http server succ")
	nod.WaitClose() // watch node exit signal
}
