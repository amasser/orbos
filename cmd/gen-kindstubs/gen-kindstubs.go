package main

import (
	"bytes"
	"flag"
	"go/format"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

type Input struct {
	Kind             string
	Package          string
	ParentDirectory  string
	QualifiedPackage string
	FileName         string
	Versions         []string
	Args             string
	Timestamp        string
}

func main() {

	kind := flag.String("kind", "", "e.g. orbiter.caos.ch/KubernetesCluster")
	parentpath := flag.String("parentpath", "", "e.g. github.com/caos/orbiter/internal/cluster-clusters")
	versions := flag.String("versions", "", "e.g. v1,v2")

	flag.Parse()
	pkg := os.Getenv("GOPACKAGE")

	input := &Input{
		Kind:             *kind,
		QualifiedPackage: filepath.Join(*parentpath, pkg),
		Package:          pkg,
		FileName:         os.Getenv("GOFILE"),
		Versions:         strings.Split(*versions, ","),
		Args:             strings.Join(os.Args[1:], " "),
		Timestamp:        time.Now().String(),
	}

	assembler := `{{ $root := . }}// Code generated by "gen-kindstubs {{ .Args }} from file {{ .FileName }}"; DO NOT EDIT.

package {{ .Package }}

import (
	"context"
	"fmt"

	"github.com/mitchellh/mapstructure"

	"github.com/caos/orbiter/internal/core/operator"

	"{{ .QualifiedPackage }}/adapter"
	"{{ .QualifiedPackage }}/model"
	{{ range .Versions }}{{ . }}builder "{{ $root.QualifiedPackage }}/model/{{ . }}"
	{{ end }}
)

type Version int

const (
	unknown Version = iota
	{{ range .Versions }}{{ . }}
	{{ end }}
)

type Kind struct {
	Kind           string
	Version 	   string
	Spec           map[string]interface{}
	Deps           map[string]map[string]interface{}
}

type assembler struct {
	path 	  []string
	overwrite func(map[string]interface{})
	builder   adapter.Builder
	built     adapter.Adapter
}

func New(configPath []string, overwrite func(map[string]interface{}), builder adapter.Builder) operator.Assembler {
	return &assembler{configPath, overwrite, builder, nil}
}

func (a *assembler) String() string { return "{{ .Kind }}" }
func (a *assembler) BuildContext() ([]string, func(map[string]interface{})) {
	return a.path, a.overwrite
}
func (a *assembler) Ensure(ctx context.Context, secrets *operator.Secrets, ensuredDependencies map[string]interface{}) (interface{}, error) {
	return a.built.Ensure(ctx, secrets, ensuredDependencies)
}
func (a *assembler) Build(serialized map[string]interface{}, nodeagentupdater operator.NodeAgentUpdater, secrets *operator.Secrets, dependant interface{}) (string, string, interface{}, map[string]operator.Assembler, error) {

	kind := &Kind{}
	if err := mapstructure.Decode(serialized, kind); err != nil {
		return "", "", nil, nil, err
	}

	if kind.Kind != "{{ .Kind }}" {
		return "", "", nil, nil, fmt.Errorf("Kind %s must be \"{{ .Kind }}\"", kind.Kind)
	}

	var spec model.UserSpec
	var subassemblersBuilder func(model.Config, map[string]map[string]interface{}) (map[string]operator.Assembler, error)
	switch kind.Version {
	{{ range .Versions }}case {{ . }}.String():
		spec, subassemblersBuilder = {{ . }}builder.Build(kind.Spec, secrets, dependant){{end}}
	default:
		return "", "", nil, nil, fmt.Errorf("Unknown version %s", kind.Version)
	}

	cfg, adapter, err := a.builder.Build(spec, nodeagentupdater)
	if err != nil {
		return "", "", nil, nil, err
	}
	a.built = adapter

	if subassemblersBuilder == nil {
		return kind.Kind, kind.Version, cfg, nil, nil
	}

	subassemblers, err := subassemblersBuilder(cfg, kind.Deps)
	if err != nil {
		return "", "", nil, nil, err
	}

	return kind.Kind, kind.Version, cfg, subassemblers, nil
}
`

	if err := generate("assembler_gen.go", assembler, input); err != nil {
		panic(err)
	}

	adapter := `{{ $root := . }}// Code generated by "gen-kindstubs {{ .Args }} from file {{ .FileName }}"; DO NOT EDIT.
	
package adapter

import (
	"context"

	"github.com/caos/orbiter/internal/core/operator"
	"{{ .QualifiedPackage }}/model"
)

type Builder interface {
	Build(model.UserSpec, operator.NodeAgentUpdater) (model.Config, Adapter, error)
}

type builderFunc func(model.UserSpec, operator.NodeAgentUpdater) (model.Config, Adapter, error)

func (b builderFunc) Build(spec model.UserSpec, nodeagent operator.NodeAgentUpdater) (model.Config, Adapter, error) {
	return b(spec, nodeagent)
}

type Adapter interface {
	Ensure(context.Context, *operator.Secrets, map[string]interface{}) (*model.Current, error)
}

type adapterFunc func(context.Context, *operator.Secrets, map[string]interface{}) (*model.Current, error)

func (a adapterFunc) Ensure(ctx context.Context, secrets *operator.Secrets, ensuredDependencies map[string]interface{}) (*model.Current, error) {
	return a(ctx, secrets, ensuredDependencies)
}
`

	if err := generate("adapter_gen.go", adapter, input, "adapter"); err != nil {
		panic(err)
	}

	builder := `{{ $root := . }}// Code generated by "gen-kindstubs {{ .Args }} from file {{ .FileName }}"; DO NOT EDIT.
	
package {{ .Package }}
	
import (
	"errors"

	"github.com/caos/orbiter/internal/core/operator"
	"{{ .QualifiedPackage }}/model"
)

var build func(map[string]interface{}, *operator.Secrets, interface{}) (model.UserSpec, func(model.Config, map[string]map[string]interface{}) (map[string]operator.Assembler, error))

func Build(spec map[string]interface{}, secrets *operator.Secrets, dependant interface{}) (model.UserSpec, func(cfg model.Config, deps map[string]map[string]interface{}) (map[string]operator.Assembler, error)) {
	if build != nil {
		return build(spec, secrets, dependant)
	}
	return model.UserSpec{}, func(_ model.Config, _ map[string]map[string]interface{}) (map[string]operator.Assembler, error){
		return nil, errors.New("Version {{ .Package }} for kind {{ .Kind }} is not yet supported")
	}
}
`

	for _, apiVersion := range input.Versions {
		input.Package = apiVersion
		if err := generate("builder_gen.go", builder, input, "model", apiVersion); err != nil {
			panic(err)
		}
	}

	if err := exec.Command("stringer", "-type", "Version").Run(); err != nil {
		panic(err)
	}
}

func generate(filename string, fileTemplate string, data interface{}, relativePathElements ...string) error {

	parsedTemplate := template.Must(template.New("").Funcs(map[string]interface{}{
		"replaceAll": strings.ReplaceAll,
	}).Parse(fileTemplate))

	var buf bytes.Buffer
	if err := parsedTemplate.Execute(&buf, data); err != nil {
		panic(err)
	}

	dir := filepath.Dir(os.Args[0])
	path := filepath.Join(append([]string{dir}, relativePathElements...)...)
	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}

	if err := os.MkdirAll(path, 0777); err != nil {
		return err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(path, filename), formatted, 0666)
}

func mkdirp(relativePathElements ...string) (path string, err error) {
	dir := filepath.Dir(os.Args[0])
	path = filepath.Join(append([]string{dir}, relativePathElements...)...)
	err = os.MkdirAll(path, 0777)
	return
}
