package static

import (
	"github.com/caos/orbiter/internal/secret"
	"github.com/caos/orbiter/internal/tree"
	"github.com/pkg/errors"

	"github.com/caos/orbiter/mntr"
)

func SecretsFunc(masterkey string) secret.Func {
	return func(monitor mntr.Monitor, desiredTree *tree.Tree) (secrets map[string]*secret.Secret, err error) {
		defer func() {
			err = errors.Wrapf(err, "building %s failed", desiredTree.Common.Kind)
		}()

		desiredKind, err := parseDesiredV0(desiredTree, masterkey)
		if err != nil {
			return nil, errors.Wrap(err, "parsing desired state failed")
		}
		desiredTree.Parsed = desiredKind

		initializeNecessarySecrets(desiredKind, masterkey)

		return getSecretsMap(desiredKind), nil
	}
}

func getSecretsMap(desiredKind *DesiredV0) map[string]*secret.Secret {
	return map[string]*secret.Secret{
		"bootstrapkeyprivate":   desiredKind.Spec.Keys.BootstrapKeyPrivate,
		"bootstrapkeypublic":    desiredKind.Spec.Keys.BootstrapKeyPublic,
		"maintenancekeyprivate": desiredKind.Spec.Keys.MaintenanceKeyPrivate,
		"maintenancekeypublic":  desiredKind.Spec.Keys.MaintenanceKeyPublic,
	}
}