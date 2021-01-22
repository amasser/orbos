package orbiter

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/caos/orbos/internal/secret"

	"github.com/caos/orbos/internal/api"
	orbconfig "github.com/caos/orbos/internal/orb"
	"github.com/caos/orbos/internal/tree"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"

	"github.com/caos/orbos/internal/git"
	"github.com/caos/orbos/internal/operator/common"
	"github.com/caos/orbos/mntr"
)

func ToEnsureResult(done bool, err error) *EnsureResult {
	return &EnsureResult{
		Err:  err,
		Done: done,
	}
}

type EnsureResult struct {
	Err  error
	Done bool
}

type ConfigureFunc func(orb orbconfig.Orb) error

func NoopConfigure(orb orbconfig.Orb) error {
	return nil
}

type QueryFunc func(nodeAgentsCurrent *common.CurrentNodeAgents, nodeAgentsDesired *common.DesiredNodeAgents, queried map[string]interface{}) (EnsureFunc, error)

type EnsureFunc func(pdf api.PushDesiredFunc) *EnsureResult

func NoopEnsure(_ api.PushDesiredFunc) *EnsureResult {
	return &EnsureResult{Done: true}
}

type retQuery struct {
	ensure EnsureFunc
	err    error
}

func QueryFuncGoroutine(query func() (EnsureFunc, error)) (EnsureFunc, error) {
	retChan := make(chan retQuery)
	go func() {
		ensure, err := query()
		retChan <- retQuery{ensure, err}
	}()
	ret := <-retChan
	return ret.ensure, ret.err
}

func EnsureFuncGoroutine(ensure func() *EnsureResult) *EnsureResult {
	retChan := make(chan *EnsureResult)
	go func() {
		retChan <- ensure()
	}()
	return <-retChan
}

type event struct {
	commit string
	files  []git.File
}

func Instrument(monitor mntr.Monitor, healthyChan chan bool) {
	healthy := true

	prometheus.MustRegister(prometheus.NewBuildInfoCollector())
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", func(writer http.ResponseWriter, request *http.Request) {
		msg := "OK"
		status := 200
		if !healthy {
			msg = "ORBITER is not healthy. See the logs."
			status = 404
		}
		writer.Write([]byte(msg))
		writer.WriteHeader(status)
	})

	go func() {
		timeout := 10 * time.Minute
		ticker := time.NewTimer(timeout)
		for {
			select {
			case newHealthiness := <-healthyChan:
				ticker.Reset(timeout)
				if newHealthiness == healthy {
					continue
				}
				healthy = newHealthiness
				if !newHealthiness {
					monitor.Error(errors.New("ORBITER is unhealthy now"))
					continue
				}
				monitor.Info("ORBITER is healthy now")
			case <-ticker.C:
				monitor.Error(errors.New("ORBITER is unhealthy now as it did not report healthiness for 10 minutes"))
				healthy = false
			}
		}
	}()

	if err := http.ListenAndServe(":9000", nil); err != nil {
		panic(err)
	}
}

func Adapt(gitClient *git.Client, monitor mntr.Monitor, finished chan struct{}, adapt AdaptFunc) (QueryFunc, DestroyFunc, ConfigureFunc, bool, *tree.Tree, *tree.Tree, map[string]*secret.Secret, error) {

	treeDesired, err := api.ReadOrbiterYml(gitClient)
	if err != nil {
		return nil, nil, nil, false, nil, nil, nil, err
	}
	treeCurrent := &tree.Tree{}

	adaptFunc := func() (QueryFunc, DestroyFunc, ConfigureFunc, bool, map[string]*secret.Secret, error) {
		return adapt(monitor, finished, treeDesired, treeCurrent)
	}
	query, destroy, configure, migrate, secrets, err := AdaptFuncGoroutine(adaptFunc)
	return query, destroy, configure, migrate, treeDesired, treeCurrent, secrets, err
}

func Takeoff(monitor mntr.Monitor, conf *Config, healthyChan chan bool) func() {

	return func() {

		var err error
		defer common.ReportHealthiness(healthyChan, err, true)

		query, _, _, migrate, treeDesired, treeCurrent, _, err := Adapt(conf.GitClient, monitor, conf.FinishedChan, conf.Adapt)
		if err != nil {
			monitor.Error(err)
			return
		}

		desiredNodeAgents := common.NodeAgentsDesiredKind{
			Kind:    "nodeagent.caos.ch/NodeAgents",
			Version: "v0",
		}
		rawDesiredNodeAgents := conf.GitClient.Read("caos-internal/orbiter/node-agents-desired.yml")
		if err = yaml.Unmarshal(rawDesiredNodeAgents, &desiredNodeAgents); err != nil {
			monitor.Error(err)
			return
		}
		desiredNodeAgents.Kind = "nodeagent.caos.ch/NodeAgents"
		desiredNodeAgents.Version = "v0"
		desiredNodeAgents.Spec.Commit = conf.OrbiterCommit

		marshalCurrentFiles := func() []git.File {
			return []git.File{{
				Path:    "caos-internal/orbiter/current.yml",
				Content: common.MarshalYAML(treeCurrent),
			}, {
				Path:    "caos-internal/orbiter/node-agents-desired.yml",
				Content: common.MarshalYAML(desiredNodeAgents),
			}}
		}
		/*
			events := make([]*event, 0)
			monitor.OnChange = mntr.Concat(func(evt string, fields map[string]string) {
				conf.PushEvents([]*ingestion.EventRequest{mntr.EventRecord("orbiter", evt, fields)})
				events = append(events, &event{
					commit: mntr.CommitRecord(mntr.AggregateCommitFields(fields)),
					files:  marshalCurrentFiles(),
				})
			}, monitor.OnChange)
		*/

		if migrate {
			if err = api.PushOrbiterYml(monitor, "Desired state migrated", conf.GitClient, treeDesired); err != nil {
				monitor.Error(err)
				return
			}
		}

		currentNodeAgents := common.NodeAgentsCurrentKind{}
		if err := yaml.Unmarshal(conf.GitClient.Read("caos-internal/orbiter/node-agents-current.yml"), &currentNodeAgents); err != nil {
			monitor.Error(err)
			return
		}

		handleAdapterError := func(err error) {
			monitor.Error(err)
			//			monitor.Error(gitClient.Clone())
			if commitErr := conf.GitClient.Commit(mntr.CommitRecord([]*mntr.Field{{Pos: 0, Key: "err", Value: err.Error()}})); commitErr != nil {
				monitor.Error(err)
				return
			}
			monitor.Error(conf.GitClient.Push())
		}

		queryFunc := func() (EnsureFunc, error) {
			return query(&currentNodeAgents.Current, &desiredNodeAgents.Spec.NodeAgents, nil)
		}
		ensure, err := QueryFuncGoroutine(queryFunc)
		if err != nil {
			handleAdapterError(err)
			return
		}

		if err := conf.GitClient.Clone(); err != nil {
			monitor.Error(err)
			return
		}

		reconciledCurrentStateMsg := "Current state reconciled"
		currentReconciled, err := conf.GitClient.StageAndCommit(mntr.CommitRecord([]*mntr.Field{{Key: "evt", Value: reconciledCurrentStateMsg}}), marshalCurrentFiles()[0])
		if err != nil {
			monitor.Error(fmt.Errorf("Commiting event \"%s\" failed: %s", reconciledCurrentStateMsg, err.Error()))
			return
		}

		if currentReconciled {
			if err := conf.GitClient.Push(); err != nil {
				monitor.Error(fmt.Errorf("Pushing event \"%s\" failed: %s", reconciledCurrentStateMsg, err.Error()))
				return
			}
		}

		//		events = make([]*event, 0)

		ensureFunc := func() *EnsureResult {
			return ensure(api.PushOrbiterDesiredFunc(conf.GitClient, treeDesired))
		}

		result := EnsureFuncGoroutine(ensureFunc)
		if result.Err != nil {
			handleAdapterError(result.Err)
			return
		}

		if result.Done {
			monitor.Info("Desired state is ensured")
		} else {
			monitor.Info("Desired state is not yet ensured")
		}
		if err := conf.GitClient.Clone(); err != nil {
			monitor.Error(fmt.Errorf("Commiting event \"%s\" failed: %s", reconciledCurrentStateMsg, err.Error()))
			return
		}

		changed, err := conf.GitClient.StageAndCommit("Current state changed", marshalCurrentFiles()...)
		if err != nil {
			monitor.Error(fmt.Errorf("commiting current state failed: %w", err))
			return
		}

		if changed {
			err = conf.GitClient.Push()
			monitor.Error(err)
		}

		/*
			for _, event := range events {

				changed, err := conf.GitClient.StageAndCommit(event.commit, event.files...)
				if err != nil {
					monitor.Error(fmt.Errorf("Commiting event \"%s\" failed: %s", event.commit, err.Error()))
					return
				}

				monitor.WithFields(map[string]interface{}{
					event.files[0].Path: string(event.files[0].Content),
					event.files[1].Path: string(event.files[1].Content),
				}).Debug("Current files staged")

				if !changed {
					panic(fmt.Sprint("Event has no effect:", event.commit))
				}
			}
			if len(events) > 0 {
				monitor.Error(conf.GitClient.Push())
			}
		*/
	}
}
