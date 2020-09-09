package v1beta2

import "github.com/caos/orbos/internal/operator/boom/api/v1beta2/k8s"

type NodeMetricsExporter struct {
	//Flag if tool should be deployed
	//@default: false
	Deploy bool `json:"deploy" yaml:"deploy"`
	//Resource requirements
	Resources *k8s.Resources `json:"resources,omitempty" yaml:"resources,omitempty"`
}
