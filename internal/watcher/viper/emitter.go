package viper

/*
import (
	"github.com/caos/orbos/mntr"
	"github.com/caos/orbos/internal/operator"
	"github.com/caos/orbos/internal/watcher"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

func New(monitor mntr.Monitor, viper *viper.Viper) operator.Watcher {
	fieldmonitor := monitor.WithFields(map[string]interface{}{
		"file": viper.ConfigFileUsed(),
	})
	return watcher.Func(func(events chan<- struct{}) error {
		viper.OnConfigChange(func(ev fsnotify.Event) {
			fieldmonitor.Debug("Configuration changed")
			events <- struct{}{}
		})
		go viper.WatchConfig()
		return nil
	})
}
*/
