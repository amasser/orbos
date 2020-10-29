package latest

import (
	"github.com/caos/orbos/internal/operator/boom/api/latest/k8s"
	"github.com/caos/orbos/internal/operator/boom/api/latest/storage"
)

type LogCollection struct {
	//Flag if tool should be deployed
	//@default: false
	Deploy bool `json:"deploy" yaml:"deploy"`
	//Fluentd Specs
	Fluentd *Fluentd `json:"fluentd,omitempty" yaml:"fluentd,omitempty"`
	//Fluentbit Specs
	Fluentbit *Component `json:"fluentbit,omitempty" yaml:"fluentbit,omitempty"`
	//Logging operator Specs
	Operator *Component `json:"operator,omitempty" yaml:"operator,omitempty"`
	//ClusterOutputs used by BOOM managed flows. BOOM managed Loki doesn't need to be specified
	ClusterOutputs []string `json:"clusterOutputs,omitempty" yaml:"clusterOutputs,omitempty"`
	//Outputs used by BOOM managed flows. BOOM managed Loki doesn't need to be specified
	Outputs []string `json:"outputs,omitempty" yaml:"outputs,omitempty"`
	//Watch namespaces other than caos-system
	WatchNamespaces []string
}

type Component struct {
	//NodeSelector for deployment
	NodeSelector map[string]string `json:"nodeSelector,omitempty" yaml:"nodeSelector,omitempty"`
	//Tolerations to run fluentbit on nodes
	Tolerations k8s.Tolerations `json:"tolerations,omitempty" yaml:"tolerations,omitempty"`
	//Resource requirements
	Resources *k8s.Resources `json:"resources,omitempty" yaml:"resources,omitempty"`
}

type Fluentd struct {
	*Component `json:",inline" yaml:",inline"`
	//Spec to define how the persistence should be handled
	PVC *storage.Spec `json:"pvc,omitempty" yaml:"pvc,omitempty"`
	//Replicas number of fluentd instances
	Replicas int
}
