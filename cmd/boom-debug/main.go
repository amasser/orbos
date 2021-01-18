package main

import (
	"context"
	"flag"
	"github.com/caos/orbos/internal/controller"
	"github.com/caos/orbos/internal/orb"
	"github.com/caos/orbos/internal/utils/clientgo"

	"github.com/caos/orbos/pkg/git"

	"github.com/caos/orbos/internal/helpers"
	"github.com/caos/orbos/internal/operator/boom"
	"github.com/caos/orbos/mntr"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

func main() {

	orbconfig := flag.String("orbconfig", "~/.orb/config", "The orbconfig file to use")
	verbose := flag.Bool("verbose", false, "Print debug levelled logs")
	metricsAddr := flag.String("metrics-addr", ":8080", "The address the metric endpoint binds to.")
	crdMode := flag.Bool("crdmode", false, "defines if the operator should run in crd mode not gitops mode")

	flag.Parse()

	monitor := mntr.Monitor{
		OnInfo:   mntr.LogMessage,
		OnChange: mntr.LogMessage,
		OnError:  mntr.LogError,
	}

	if *verbose {
		monitor = monitor.Verbose()
	}

	if *crdMode {
		clientgo.InConfig = false
		_, err := orb.ParseOrbConfig(helpers.PruneHome(*orbconfig))
		if err != nil {
			panic(err)
		}

		if err := controller.Start(monitor, "crdoperators", "./artifacts", *metricsAddr, controller.Boom); err != nil {
			panic(err)
		}
	} else {

		ensure := git.New(context.Background(), monitor.WithField("task", "ensure"), "Boom", "boom@caos.ch")
		query := git.New(context.Background(), monitor.WithField("task", "query"), "Boom", "boom@caos.ch")

		ensure.Clone()
		query.Clone()

		takeoff, _ := boom.Takeoff(
			monitor,
			"./artifacts",
			true,
			helpers.PruneHome(*orbconfig),
			ensure, query,
		)

		for {
			takeoff()
		}
	}
}
