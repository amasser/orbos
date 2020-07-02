package iam

import (
	"github.com/caos/orbos/internal/operator/zitadel/kinds/iam/configuration"
	"github.com/caos/orbos/internal/tree"
	"github.com/pkg/errors"
)

type DesiredV0 struct {
	Common   *tree.Common `yaml:",inline"`
	Spec     Spec
	Database *tree.Tree
}

type Spec struct {
	Verbose       bool
	ReplicaCount  int `yaml:"replicaCount,omitempty"`
	Version       string
	Configuration *configuration.Configuration
}

func parseDesiredV0(desiredTree *tree.Tree) (*DesiredV0, error) {
	desiredKind := &DesiredV0{
		Common: desiredTree.Common,
		Spec:   Spec{},
	}

	if err := desiredTree.Original.Decode(desiredKind); err != nil {
		return nil, errors.Wrap(err, "parsing desired state failed")
	}

	return desiredKind, nil
}
