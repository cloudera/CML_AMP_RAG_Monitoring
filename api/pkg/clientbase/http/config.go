package cbhttp

import (
	"time"

	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
)

type Config struct {
	Timeout        time.Duration `env:"CLIENT_HTTP_TIMEOUT"`
	AvoidRedirects bool          `env:"CLIENT_HTTP_AVOID_REDIRECTS"`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
