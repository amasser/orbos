package api

import (
	"github.com/caos/orbos/internal/operator/boom/api/common"
	"github.com/caos/orbos/internal/operator/boom/api/latest"
	"github.com/caos/orbos/internal/operator/boom/api/migrate"
	"github.com/caos/orbos/internal/operator/boom/api/v1beta1"
	"github.com/caos/orbos/internal/operator/boom/api/v1beta2"
	"github.com/caos/orbos/internal/secret"
	"github.com/caos/orbos/internal/tree"
	"github.com/caos/orbos/pkg/labels"
	"github.com/pkg/errors"
)

func ParseToolset(desiredTree *tree.Tree, l *labels.Operator) (*labels.API, *latest.Toolset, bool, map[string]*secret.Secret, error) {
	desiredKindCommon := common.New()
	if err := desiredTree.Original.Decode(desiredKindCommon); err != nil {
		return nil, nil, false, nil, errors.Wrap(err, "parsing desired state failed")
	}

	kind := "Toolset"

	switch desiredKindCommon.APIVersion {
	case "boom.caos.ch/v1beta1":
		old, _, err := v1beta1.ParseToolset(desiredTree)
		if err != nil {
			return nil, nil, false, nil, err
		}
		v1beta2Toolset, _ := migrate.V1beta1Tov1beta2(old)
		v1Toolset, secrets := migrate.V1beta2Tov1(v1beta2Toolset)

		return labels.MustForAPI(l, kind, "v1beta1"), v1Toolset, true, secrets, err
	case "boom.caos.ch/v1beta2":
		v1beta2Toolset, _, err := v1beta2.ParseToolset(desiredTree)
		if err != nil {
			return nil, nil, false, nil, err
		}
		v1Toolset, secrets := migrate.V1beta2Tov1(v1beta2Toolset)

		return labels.MustForAPI(l, kind, "v1beta2"), v1Toolset, true, secrets, nil
	case "boom.caos.ch/v1":
		desiredKind, secrets, err := latest.ParseToolset(desiredTree)
		if err != nil {
			return nil, nil, false, nil, err
		}
		return labels.MustForAPI(l, kind, "v1"), desiredKind, false, secrets, nil
	default:
		return nil, nil, false, nil, errors.New("APIVersion unknown")
	}
}

/*
func SecretsFunc() secret.Func {
	return func(monitor mntr.Monitor, desiredTree *tree.Tree) (secrets map[string]*secret.Secret, err error) {
		desiredKindCommon := common.New()
		if err := desiredTree.Original.Decode(desiredKindCommon); err != nil {
			return nil, errors.Wrap(err, "parsing desired state failed")
		}

		switch desiredKindCommon.APIVersion {
		case "boom.caos.ch/v1beta1":
			return v1beta1.SecretsFunc(desiredTree)
		case "boom.caos.ch/v1beta2":
			return v1beta2.SecretsFunc(desiredTree)
		default:
			return nil, errors.New("APIVersion unknown")
		}
	}
}*/
