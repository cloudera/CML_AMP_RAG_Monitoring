package datasource

import (
	"errors"
	log "github.com/sirupsen/logrus"
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
)

type Config struct {
	LocalMLFlowBaseUrl string `env:"LOCAL_MLFLOW_BASE_URL" envDefault:"http://localhost:5000"`
	CDSWDomain         string `env:"CDSW_DOMAIN" envDefault:""`
	CDSWApiProtocol    string `env:"CDSW_API_PROTOCOL" envDefault:"https"`
	CDSWMLFlowBaseUrl  string `env:"CDSW_API_URL" envDefault:""`
	CDSWProjectID      string `env:"CDSW_PROJECT_ID" envDefault:""`
	CDSWApiKey         string `env:"CDSW_APIV2_KEY" envDefault:""`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	if cfg.CDSWMLFlowBaseUrl == "" && cfg.CDSWDomain == "" {
		return nil, errors.New("one of CDSWDomain or CDSW_API_URL is required")
	}
	if cfg.CDSWMLFlowBaseUrl == "" {
		cfg.CDSWMLFlowBaseUrl = cfg.CDSWApiProtocol + "://" + cfg.CDSWDomain
	}
	if cfg.CDSWProjectID == "" {
		return nil, errors.New("CDSW_PROJECT_ID is required")
	}
	if cfg.CDSWApiKey == "" {
		return nil, errors.New("CDSW_APIV2_KEY is required")
	}
	log.Printf("CDSW base url: %s", cfg.CDSWMLFlowBaseUrl)
	log.Printf("CDSW project ID: %s", cfg.CDSWProjectID)
	log.Printf("CDSW api key is %d tokens in length", len(cfg.CDSWApiKey))
	return &cfg, nil
}
