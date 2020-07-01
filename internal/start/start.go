package start

import (
	"context"
	"errors"
	"runtime/debug"
	"strings"
	"time"

	"github.com/caos/orbos/internal/api"
	"github.com/caos/orbos/internal/executables"
	"github.com/caos/orbos/internal/git"
	"github.com/caos/orbos/internal/ingestion"
	"github.com/caos/orbos/internal/operator/boom"
	"github.com/caos/orbos/internal/operator/orbiter"
	"github.com/caos/orbos/internal/operator/orbiter/kinds/orb"
	"github.com/caos/orbos/internal/operator/secretfuncs"
	orbconfig "github.com/caos/orbos/internal/orb"
	"github.com/caos/orbos/internal/secret"
	"github.com/caos/orbos/internal/utils/orbgit"
	"github.com/caos/orbos/mntr"
	"github.com/golang/protobuf/ptypes"
	structpb "github.com/golang/protobuf/ptypes/struct"
	"google.golang.org/grpc"
)

type OrbiterConfig struct {
	Recur            bool
	Destroy          bool
	Deploy           bool
	Verbose          bool
	Version          string
	OrbConfigPath    string
	GitCommit        string
	IngestionAddress string
}

func Orbiter(ctx context.Context, monitor mntr.Monitor, conf *OrbiterConfig, orbctlGit *git.Client) ([]string, error) {

	finishedChan := make(chan struct{})

	go func() {
		takeoffChan := make(chan struct{})
		go func() {
			takeoffChan <- struct{}{}
		}()

		var initialized bool
		for range takeoffChan {
			iterate(conf, !initialized, ctx, monitor, finishedChan, takeoffChan)
			initialized = true
		}
	}()

	<-finishedChan

	orbFile, err := orbconfig.ParseOrbConfig(conf.OrbConfigPath)
	if err != nil {
		return nil, err
	}

	if err := orbctlGit.Clone(); err != nil {
		return nil, err
	}

	return GetKubeconfigs(monitor, orbctlGit, orbFile)
}

func iterate(conf *OrbiterConfig, firstIteration bool, ctx context.Context, monitor mntr.Monitor, finishedChan chan struct{}, takeoffChan chan struct{}) {
	orbFile, err := orbconfig.ParseOrbConfig(conf.OrbConfigPath)
	if err != nil {
		panic(err)
	}

	pushEvents := func(events []*ingestion.EventRequest) error { return nil }
	if conf.IngestionAddress != "" {
		conn, err := grpc.Dial(conf.IngestionAddress, grpc.WithInsecure())
		if err != nil {
			panic(err)
		}

		ingc := ingestion.NewIngestionServiceClient(conn)
		defer conn.Close()

		pushEvents = func(events []*ingestion.EventRequest) error {
			_, err := ingc.PushEvents(ctx, &ingestion.EventsRequest{
				Orb:    orbFile.URL,
				Events: events,
			})
			return err
		}
	}

	if firstIteration {
		if conf.Recur {
			orbiter.Metrics()
		}

		if err := pushEvents([]*ingestion.EventRequest{{
			CreationDate: ptypes.TimestampNow(),
			Type:         "orbiter.tookoff",
			Data: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"commit": &structpb.Value{Kind: &structpb.Value_StringValue{StringValue: conf.GitCommit}},
				},
			},
		}}); err != nil {
			panic(err)
		}

		started := float64(time.Now().UTC().Unix())

		go func() {
			for range time.Tick(time.Minute) {
				pushEvents([]*ingestion.EventRequest{{
					CreationDate: ptypes.TimestampNow(),
					Type:         "orbiter.running",
					Data: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"since": {Kind: &structpb.Value_NumberValue{NumberValue: started}},
						},
					},
				}})
			}
		}()

		executables.Populate()

		monitor.WithFields(map[string]interface{}{
			"version": conf.Version,
			"commit":  conf.GitCommit,
			"destroy": conf.Destroy,
			"verbose": conf.Verbose,
			"repoURL": orbFile.URL,
		}).Info("Orbiter took off")
	}

	gitClientConf := &orbgit.Config{
		Comitter:  "orbiter",
		Email:     "orbiter@caos.ch",
		OrbConfig: orbFile,
		Action:    "iteration",
	}

	gitClient, cleanUp, err := orbgit.NewGitClient(ctx, monitor, gitClientConf, true)
	if err != nil {
		monitor.Error(err)
		finishedChan <- struct{}{}
	}

	adaptFunc := orb.AdaptFunc(
		orbFile,
		conf.GitCommit,
		!conf.Recur,
		conf.Deploy)

	takeoffConf := &orbiter.Config{
		OrbiterCommit: conf.GitCommit,
		GitClient:     gitClient,
		Adapt:         adaptFunc,
		FinishedChan:  finishedChan,
		PushEvents:    pushEvents,
	}

	takeoff := orbiter.Takeoff(monitor, takeoffConf)

	go func() {
		started := time.Now()
		takeoff()

		monitor.WithFields(map[string]interface{}{
			"took": time.Since(started),
		}).Info("Iteration done")
		debug.FreeOSMemory()
		takeoffChan <- struct{}{}
		cleanUp()
	}()
}

func GetKubeconfigs(monitor mntr.Monitor, gitClient *git.Client, orbFile *orbconfig.Orb) ([]string, error) {
	kubeconfigs := make([]string, 0)

	orbTree, err := api.ReadOrbiterYml(gitClient)
	if err != nil {
		return nil, errors.New("Failed to parse orbiter.yml")
	}

	orbDef, err := orb.ParseDesiredV0(orbTree)
	if err != nil {
		return nil, errors.New("Failed to parse orbiter.yml")
	}

	for clustername, _ := range orbDef.Clusters {
		path := strings.Join([]string{"orbiter", clustername, "kubeconfig"}, ".")

		value, err := secret.Read(
			monitor,
			gitClient,
			secretfuncs.GetSecrets(),
			path)
		if err != nil || value == "" {
			return nil, errors.New("Failed to get kubeconfig")
		}
		monitor.Info("Read kubeconfig for boom deployment")

		kubeconfigs = append(kubeconfigs, value)
	}

	return kubeconfigs, nil
}

func Boom(monitor mntr.Monitor, orbConfigPath string, localmode bool, version string) error {
	boom.Metrics(monitor)

	takeoffChan := make(chan struct{})
	go func() {
		takeoffChan <- struct{}{}
	}()

	for range takeoffChan {
		orbConfig, err := orbconfig.ParseOrbConfig(orbConfigPath)
		if err != nil {
			monitor.Error(err)
			return err
		}

		boomChan := make(chan struct{})
		currentChan := make(chan struct{})

		takeoff, takeoffCurrent := boom.Takeoff(
			monitor,
			orbConfig,
			"/boom",
			localmode,
			version,
		)
		go func() {
			started := time.Now()
			takeoffCurrent()

			monitor.WithFields(map[string]interface{}{
				"took": time.Since(started),
			}).Info("Iteration done")
			debug.FreeOSMemory()

			currentChan <- struct{}{}
		}()
		go func() {
			started := time.Now()
			takeoff()

			monitor.WithFields(map[string]interface{}{
				"took": time.Since(started),
			}).Info("Iteration done")
			debug.FreeOSMemory()

			boomChan <- struct{}{}
		}()

		go func() {
			<-currentChan
			<-boomChan

			takeoffChan <- struct{}{}
		}()
	}

	return nil
}
