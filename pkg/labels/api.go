package labels

import (
	"errors"

	"gopkg.in/yaml.v3"
)

var (
	_ Labels = (*API)(nil)
)

type API struct {
	model InternalAPI
	base  *Operator
}

func ForAPI(l *Operator, kind, version string) (*API, error) {
	if kind == "" || version == "" {
		return nil, errors.New("kind and version must not be nil")
	}

	return &API{
		base: l,
		model: InternalAPI{
			Kind:             kind,
			ApiVersion:       version,
			InternalOperator: l.model,
		},
	}, nil
}

func MustForAPI(l *Operator, kind, version string) *API {
	a, err := ForAPI(l, kind, version)
	if err != nil {
		panic(err)
	}
	return a
}

func (l *API) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&l.model); err != nil {
		return err
	}
	l.base = &Operator{}
	return node.Decode(l.base)
}

func (l *API) Major() int8 {
	return l.base.Major()
}

func (l *API) Equal(r comparable) bool {
	if right, ok := r.(*API); ok {
		return l.model == right.model
	}
	return false
}

func (l *API) MarshalYAML() (interface{}, error) {
	return nil, errors.New("type *labels.API is not serializable")
}

type InternalAPI struct {
	Kind             string `yaml:"caos.ch/kind"`
	ApiVersion       string `yaml:"caos.ch/apiversion"`
	InternalOperator `yaml:",inline"`
}
