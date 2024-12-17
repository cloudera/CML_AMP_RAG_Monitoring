package runs

import (
	"fmt"
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	"time"
)

type Config struct {
	Enabled           bool          `env:"MLFLOW_RECONCILER_ENABLED" envDefault:"true"`
	ResyncFrequency   time.Duration `env:"MLFLOW_RECONCILER_RESYNC_FREQUENCY" envDefault:"15s"`
	GCResyncFrequency time.Duration `env:"MLFLOW_RECONCILER_GC_RESYNC_FREQUENCY" envDefault:"1m"`
	ResyncMaxItems    int           `env:"MLFLOW_RECONCILER_RESYNC_MAX_ITEMS" envDefault:"1000"`
	MaxWorkers        int           `env:"MLFLOW_RECONCILER_MAX_WORKERS" envDefault:"1"`
	RunMaxItems       int           `env:"MLFLOW_RECONCILER_RUN_MAX_ITEMS" envDefault:"1"`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	err = validateConfig(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

var ErrInvalidGCResyncFrequency = fmt.Errorf("invalid garbage collection resync frequency")
var ErrInvalidResyncMaxItems = fmt.Errorf("invalid resync max items")

func validateConfig(config *Config) error {
	if config.ResyncFrequency < 1*time.Second {
		return reconciler.ErrInvalidResyncFrequency
	}

	if config.MaxWorkers < 1 {
		return reconciler.ErrInvalidMaxWorkers
	}

	if config.RunMaxItems < 1 {
		return reconciler.ErrInvalidRunMaxItems
	}

	if config.GCResyncFrequency < 1*time.Second {
		return ErrInvalidGCResyncFrequency
	}
	if config.ResyncMaxItems < 1 {
		return ErrInvalidResyncMaxItems
	}
	return nil
}
