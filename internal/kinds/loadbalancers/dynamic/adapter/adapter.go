//go:generate goderive .

package adapter

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"text/template"

	"github.com/pkg/errors"

	"github.com/caos/orbiter/internal/core/helpers"
	"github.com/caos/orbiter/internal/core/operator"
	"github.com/caos/orbiter/internal/kinds/clusters/core/infra"
	"github.com/caos/orbiter/internal/kinds/loadbalancers/dynamic/model"
	"github.com/caos/orbiter/internal/kinds/providers/core"
)

type Overwriter interface {
	Overwrite() model.UserSpec
}

type Data struct {
	VIPs       []model.VIP
	RemoteUser string
	State      string
	RouterID   int
	Self       infra.Compute
	Peers      []infra.Compute
}

func New(remoteUser string) Builder {
	return builderFunc(func(spec model.UserSpec, _ operator.NodeAgentUpdater) (model.Config, Adapter, error) {

		cfg := model.Config{}

		return cfg, adapterFunc(func(ctx context.Context, secrets *operator.Secrets, deps map[string]interface{}) (*model.Current, error) {

			for depName, dep := range deps {
				overwriter, ok := dep.(Overwriter)
				if !ok {
					return nil, errors.Errorf("Unknown dependency %s", depName)
				}
				for key, value := range overwriter.Overwrite() {
					spec[key] = append(spec[key], value...)
				}
			}

			if err := spec.Validate(); err != nil {
				return nil, err
			}

			sourcePools := make(map[string][]string)
			addresses := make(map[string]string)
			for _, pool := range spec {
				for _, vip := range pool {
					for _, src := range vip.Transport {
						addresses[src.Name] = fmt.Sprintf("%s:%d", vip.IP, src.SourcePort)
					destinations:
						for _, dest := range src.Destinations {
							if _, ok := sourcePools[dest.Pool]; !ok {
								sourcePools[dest.Pool] = make([]string, 0)
							}
							for _, existing := range sourcePools[dest.Pool] {
								if dest.Pool == existing {
									continue destinations
								}
							}
							sourcePools[dest.Pool] = append(sourcePools[dest.Pool], dest.Pool)
						}
					}
				}
			}

			return &model.Current{
				Addresses:   addresses,
				SourcePools: sourcePools,
				Desire: func(pool string, changesAllowed bool, svc core.ComputesService, nodeagent func(infra.Compute) *operator.NodeAgentCurrent) error {

					vips, ok := spec[pool]
					if !ok {
						return nil
					}

					computes, err := svc.List(pool, true)
					if err != nil {
						return err
					}

					computesData := make([]Data, len(computes))
					for idx, compute := range computes {
						computesData[idx] = Data{
							RemoteUser: remoteUser,
							VIPs:       vips,
							Self:       compute,
							Peers: deriveFilter(func(cmp infra.Compute) bool {
								return cmp.ID() != compute.ID()
							}, append([]infra.Compute(nil), []infra.Compute(computes)...)),
							State: "BACKUP",
						}
						if idx == 0 {
							computesData[idx].State = "MASTER"
						}
					}

					templateFuncs := template.FuncMap(map[string]interface{}{
						"computes": svc.List,
						"add": func(i, y int) int {
							return i + y
						},
						"healthcmd": vrrpHealthChecksScript,
						//						"upstreamHealthchecks": deriveFmap(vip model.VIP) []string {
						//							return deriveFmap(func(src model.Source) []string {
						//
						//								if src.HealthChecks != nil {
						//									return fmt.Sprintf(check, src.HealthChecks.Protocol)
						//								}
						//							}, vip.Transport)
						//						},
					})

					keepaliveDTemplate := template.Must(template.New("").Funcs(templateFuncs).Parse(`{{ $root := . }}global_defs {
    enable_script_security
    script_user {{ $root.RemoteUser }}
}

vrrp_sync_group VG1 {
    group {
{{ range $idx, $_ := .VIPs }}        VI_{{ $idx }}
{{ end }}    }
}

{{ range $idx, $vip := .VIPs }}vrrp_script chk_{{ $vip.IP }} {
    script       "{{ healthcmd $vip.Transport }}"
    interval 2   # check every 2 seconds
    fall 2       # require 2 failures for KO
    rise 2       # require 2 successes for OK
}

vrrp_instance VI_{{ $idx }} {
    state {{ $root.State }}
    unicast_src_ip {{ $root.Self.InternalIP }}
    unicast_peer {
        {{ range $peer := $root.Peers }}{{ $peer.InternalIP }}
        {{ end }}    }
    interface eth0
    virtual_router_id {{ add 55 $idx }}
    advert_int 1
    authentication {
        auth_type PASS
        auth_pass [ REDACTED ]
    }
    virtual_ipaddress {
        {{ $vip.IP }}
    }
    track_script {
        chk_{{ $vip.IP }}
    }
}
{{ end }}
`))

					//					nginxTemplate := template.Must(template.New("").Funcs(templateFuncs).Parse(`{{ $root := . }}stream { {{ range $vip := .VIPs }}{{ range $src := $vip.Transport }}
					//	upstream {{ $src.Name }} {		{{ range $dest := $src.Destinations }}{{ range $compute := computes $dest.Pool }}
					//		server {{ $compute.InternalIP }}:{{ $dest.Port }}; # {{ $dest.Pool }}{{end}}{{ end }}
					//	}
					//	server {
					//		listen {{ $vip.IP }}:{{ $src.SourcePort }};
					//		proxy_pass {{ $src.Name }};
					//	}
					//{{ end }}{{ end }}}`))

					var wg sync.WaitGroup
					synchronizer := helpers.NewSynchronizer(&wg)

					for _, d := range computesData {
						wg.Add(1)

						go parse(synchronizer, keepaliveDTemplate, d, nodeagent(d.Self), func(result string, na *operator.NodeAgentCurrent) {
							pkg := operator.Package{Config: map[string]string{"keepalived.conf": result}}
							if changesAllowed && !na.Software.KeepaliveD.Equals(&pkg) {
								na.AllowChanges()
							}
							na.DesireSoftware(&operator.Software{KeepaliveD: pkg})
						})
						//						go parse(synchronizer, nginxTemplate, d, func(cfg string) {
						//							key := "nginx.conf"
						//							if na.Software.Nginx.Config == nil || na.Software.Nginx.Config[key] != cfg {
						//								na.AllowChanges()
						//							}
						//							na.DesireSoftware(&operator.Software{Nginx: operator.Package{Config: map[string]string{key: cfg}}})
						//						})
					}

					wg.Wait()

					if synchronizer.IsError() {
						return synchronizer
					}
					return nil
				},
			}, nil
		}), nil
	})
}

func parse(synchronizer *helpers.Synchronizer, parsedTemplate *template.Template, computesData Data, na *operator.NodeAgentCurrent, then func(string, *operator.NodeAgentCurrent)) {

	var buf bytes.Buffer
	err := parsedTemplate.Execute(&buf, computesData)
	if err != nil {
		synchronizer.Done(err)
		return
	}

	then(buf.String(), na)

	synchronizer.Done(nil)
}

func toChan(templates []*template.Template) <-chan *template.Template {
	ch := make(chan *template.Template)
	go func() {
		for _, template := range templates {
			ch <- template
		}
		close(ch)
	}()
	return ch
}