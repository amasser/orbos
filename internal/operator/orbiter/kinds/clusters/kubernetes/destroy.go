package kubernetes

import (
	"github.com/caos/orbos/internal/operator/orbiter/kinds/clusters/core/infra"
	"github.com/caos/orbos/pkg/kubernetes"
)

func destroy(providerCurrents map[string]interface{}, k8sClient *kubernetes.Client) error {

	if k8sClient.Available() {
		k8sClient.DeleteDeployment("caos-system", "orbiter")
	}

	for _, provider := range providerCurrents {
		prov := provider.(infra.ProviderCurrent)
		for _, pool := range prov.Pools() {
			machines, err := pool.GetMachines()
			if err != nil {
				return err
			}
			for _, machine := range machines {
				machine.Remove()
			}
		}
	}
	return nil
}
