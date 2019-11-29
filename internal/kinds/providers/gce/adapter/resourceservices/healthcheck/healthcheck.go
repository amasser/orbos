package healthcheck

import (
	"errors"

	"github.com/caos/infrop/internal/core/logging"
	"github.com/caos/infrop/internal/kinds/providers/core"

	"github.com/caos/infrop/internal/kinds/providers/gce/edge/api"
	"github.com/caos/infrop/internal/kinds/providers/gce/model"
	"google.golang.org/api/compute/v1"
)

type hc struct {
	logger logging.Logger
	spec   *model.UserSpec
	svc    *compute.HealthChecksService
	caller *api.Caller
}

type Config struct {
	Port int64
	Path string
}

func New(logger logging.Logger, svc *compute.Service, spec *model.UserSpec, caller *api.Caller) core.ResourceService {
	return &hc{
		logger: logger.WithFields(map[string]interface{}{"type": "health check"}),
		spec:   spec,
		svc:    compute.NewHealthChecksService(svc),
		caller: caller,
	}
}

func (h *hc) Abbreviate() string {
	return "hc"
}

func (h *hc) Desire(payload interface{}) (interface{}, error) {
	cfg, ok := payload.(*Config)
	if !ok {
		return nil, errors.New("Config must be of type *healthcheck.Config")
	}

	return &compute.HealthCheck{
		Type: "HTTPS",
		HttpsHealthCheck: &compute.HTTPSHealthCheck{
			Port:        cfg.Port,
			RequestPath: cfg.Path,
		},
	}, nil
}

type Ensured struct {
	URL string
}

func (h *hc) Ensure(id string, desired interface{}, dependencies []interface{}) (interface{}, error) {

	logger := h.logger.WithFields(map[string]interface{}{"name": id})

	// ID validations
	if len(dependencies) > 0 {
		return nil, errors.New("Healthchecks can't have dependencies")
	}

	selflink, err := h.caller.GetResourceSelfLink(id, []interface{}{
		h.svc.Get(h.spec.Project, id),
	})
	if err != nil {
		return nil, err
	}

	if selflink != nil {
		return &Ensured{*selflink}, nil
	}

	hc := *desired.(*compute.HealthCheck)
	hc.Name = id

	op, err := h.caller.RunFirstSuccessful(
		logger,
		api.Insert,
		h.svc.Insert(h.spec.Project, &hc))
	if err != nil {
		return nil, err
	}
	return &Ensured{op.TargetLink}, nil
}

func (h *hc) Delete(id string) error {
	logger := h.logger.WithFields(map[string]interface{}{"name": id})
	_, err := h.caller.RunFirstSuccessful(logger, api.Delete, h.svc.Delete(h.spec.Project, id))
	return err
}

func (h *hc) AllExisting() ([]string, error) {
	return h.caller.ListResources(h, []interface{}{
		h.svc.List(h.spec.Project),
	})
}
