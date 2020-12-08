package client

import (
	"strings"

	"github.com/caos/orbos/pkg/labels"

	"github.com/caos/orbos/pkg/kubernetes"
)

func QueryCertificates(
	namespace string,
	componentLabels *labels.Component,
	k8sClient kubernetes.ClientInt,
) (
	[]string,
	error,
) {

	list, err := k8sClient.ListSecrets(namespace, labels.MustK8sMap(componentLabels))
	if err != nil {
		return nil, err
	}
	certs := []string{}
	for _, secret := range list.Items {
		if strings.HasPrefix(secret.Name, clientSecretPrefix) {
			certs = append(certs, strings.TrimPrefix(secret.Name, clientSecretPrefix))
		}
	}

	return certs, nil
}
