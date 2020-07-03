package iam

import (
	"github.com/caos/orbos/internal/secret"
	"github.com/caos/orbos/internal/tree"
	"github.com/caos/orbos/mntr"
	"github.com/pkg/errors"
)

func SecretsFunc() secret.Func {
	return func(monitor mntr.Monitor, desiredTree *tree.Tree) (secrets map[string]*secret.Secret, err error) {
		defer func() {
			err = errors.Wrapf(err, "building %s failed", desiredTree.Common.Kind)
		}()

		desiredKind, err := parseDesiredV0(desiredTree)
		if err != nil {
			return nil, errors.Wrap(err, "parsing desired state failed")
		}
		desiredTree.Parsed = desiredKind

		return getSecretsMap(desiredKind), nil
	}
}

func getSecretsMap(desiredKind *DesiredV0) map[string]*secret.Secret {
	return map[string]*secret.Secret{
		"consoleenvironmentjson": desiredKind.Spec.Configuration.ConsoleEnvironmentJSON,
		"serviceaccountjson":     desiredKind.Spec.Configuration.Tracing.ServiceAccountJSON,
		"keys":                   desiredKind.Spec.Configuration.Secrets.Keys,
		"googlechaturl":          desiredKind.Spec.Configuration.Notifications.GoogleChatURL,
		"twiliosid":              desiredKind.Spec.Configuration.Notifications.Twilio.SID,
		"twilioauthtoken":        desiredKind.Spec.Configuration.Notifications.Twilio.AuthToken,
		"emailappkey":            desiredKind.Spec.Configuration.Notifications.Email.AppKey,
	}
}
