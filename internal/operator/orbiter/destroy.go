package orbiter

import (
	"github.com/caos/orbos/internal/api"
	"github.com/caos/orbos/internal/operator/common"
	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/pkg/git"
	"github.com/caos/orbos/pkg/tree"
)

type DestroyFunc func(map[string]interface{}) error

func NoopDestroy(map[string]interface{}) error {
	return nil
}

func DestroyFuncGoroutine(query func() error) error {
	retChan := make(chan error)
	go func() {
		retChan <- query()
	}()
	return <-retChan
}

func Destroy(monitor mntr.Monitor, gitClient *git.Client, adapt AdaptFunc, finishedChan chan struct{}) error {
	treeDesired, err := api.ReadOrbiterYml(gitClient)
	if err != nil {
		return err
	}

	treeCurrent := &tree.Tree{}

	_, destroy, _, _, _, err := adapt(monitor, finishedChan, treeDesired, treeCurrent)
	if err != nil {
		return err
	}

	if err := destroy(make(map[string]interface{})); err != nil {
		return err
	}

	monitor.OnChange = func(evt string, fields map[string]string) {
		if err := gitClient.UpdateRemote(mntr.CommitRecord([]*mntr.Field{{Key: "evt", Value: evt}}), git.File{
			Path:    "caos-internal/orbiter/current.yml",
			Content: []byte(""),
		}, git.File{
			Path:    "caos-internal/orbiter/node-agents-current.yml",
			Content: []byte(""),
		}, git.File{
			Path:    "caos-internal/orbiter/node-agents-desired.yml",
			Content: []byte(""),
		}, git.File{
			Path:    "orbiter.yml",
			Content: common.MarshalYAML(treeDesired),
		}); err != nil {
			panic(err)
		}
	}
	monitor.Changed("Orb destroyed")
	return nil
}
