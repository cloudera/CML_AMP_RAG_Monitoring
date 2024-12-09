package datasource

import (
	"errors"
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
)

type Config struct {
	LocalMLFlowBaseUrl  string `env:"LOCAL_MLFLOW_BASE_URL" envDefault:"http://localhost:5000"`
	RemoteMLFlowBaseUrl string `env:"CDSW_API_URL" envDefault:""`
	RemoteProjectID     string `env:"CDSW_PROJECT_NUM" envDefault:""`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	if cfg.RemoteMLFlowBaseUrl == "" {
		return nil, errors.New("CDSW_API_URL is required")
	}
	if cfg.RemoteProjectID == "" {
		return nil, errors.New("CDSW_PROJECT_NUM is required")
	}

	return &cfg, nil
}
