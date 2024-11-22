package datasource

import lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"

type Config struct {
	MLFlowBaseUrl string `env:"MLFLOW_BASE_URL" envDefault:"http://localhost:5000"`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
