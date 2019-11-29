package immediate

import (
	"github.com/caos/orbiter/internal/core/logging"
	"github.com/caos/orbiter/internal/core/operator"
	"github.com/caos/orbiter/internal/edge/watcher"
)

func New(logger logging.Logger) operator.Watcher {
	return watcher.Func(func(changes chan<- struct{}) error {
		logger.Debug("Immediate triggered")
		go func() {
			changes <- struct{}{}
		}()
		return nil
	})
}
