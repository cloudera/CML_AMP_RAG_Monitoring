package config

import (
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
)

type HeaderEndpoints map[string]string

type Config struct {
	lconfig.PodInfo
	Migrate          bool   `env:"MIGRATE" envDefault:"true"`
	MigrationVersion *uint  `env:"MIGRATION_VERSION"`
	CDSWProjectID    string `env:"CDSW_PROJECT_ID" envDefault:""`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
