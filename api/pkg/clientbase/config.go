package clientbase

import (
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
)

type Config struct {
	SwaggerAddress string `env:"SWAGGER_ADDRESS"`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
