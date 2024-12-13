package datasource

import (
	"errors"
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
)

type Config struct {
	LocalMLFlowBaseUrl string `env:"LOCAL_MLFLOW_BASE_URL" envDefault:"http://localhost:5000"`
	CDSWMLFlowBaseUrl  string `env:"CDSW_API_URL" envDefault:""`
	CDSWProjectID      string `env:"CDSW_PROJECT_ID" envDefault:""`
	CDSWApiKey         string `env:"CDSW_API_KEY" envDefault:""`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	if cfg.CDSWMLFlowBaseUrl == "" {
		return nil, errors.New("CDSW_API_URL is required")
	}
	if cfg.CDSWProjectID == "" {
		return nil, errors.New("CDSW_PROJECT_ID is required")
	}
	if cfg.CDSWApiKey == "" {
		return nil, errors.New("CDSW_API_KEY is required")
	}

	return &cfg, nil
}
