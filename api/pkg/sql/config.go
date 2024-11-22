package lsql

import (
	"fmt"
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
)

type Config struct {
	ConfigSecrets

	Engine         string        `env:"SQL_DB_ENGINE" envDefault:"sqlite"`
	DatabaseName   string        `env:"SQL_DB_NAME"`
	Address        string        `env:"SQL_DB_ADDRESS" envDefault:"localhost:4000"`
	Domain         string        `env:"SQL_DB_DOMAIN"`
	Options        string        `env:"SQL_DB_OPTIONS" envDefault:"parseTime=true&maxAllowedPacket=0&multiStatements=true&clientFoundRows=true"`
	DisableMetrics bool          `env:"SQL_DB_DISABLE_METRICS"`
	MaxLifetime    time.Duration `env:"SQL_DB_MAX_LIFETIME" envDefault:"30m"`
	MaxIdleConns   int           `env:"SQL_DB_MAX_IDLE_CONNS" envDefault:"5"`
	MaxOpenConns   int           `env:"SQL_DB_MAX_OPEN_CONNS" envDefault:"20"`
	ConfigLocation string        `env:"SQL_DB_CONFIG_LOCATION"`
	FilePathRoot   string        `env:"FILE_PATH_ROOT" envDefault:"^/home/jenkins/agent/workspace/build/[^/]+/"`
}

type ConfigSecrets struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	if cfg.ConfigLocation != "" {
		err = cfg.loadFile()
		if err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

func (cfg *Config) PartialAddress() string {
	var connString string
	// NTLM authentication prepends the domain to username
	usernameAndMaybeDomain := cfg.Username
	switch strings.ToLower(cfg.Engine) {
	case "mysql":
		connString = "%s:%s@tcp(%s)/"
	case "sqlserver":
		connString = "sqlserver://%s:%s@%s"
		if cfg.Domain != "" {
			usernameAndMaybeDomain = fmt.Sprintf("%s%%5C%s", cfg.Domain, cfg.Username)
		}
	case "sqlite":
		if cfg.Address != "" {
			return cfg.Address
		}
		return ":memory:"
	default:
		return ""
	}
	return fmt.Sprintf(
		connString,
		usernameAndMaybeDomain,
		cfg.Password,
		cfg.Address,
	)
}

func (cfg *Config) FullAddress() string {
	var connString string
	// NTLM authentication prepends the domain to username
	usernameAndMaybeDomain := cfg.Username
	switch strings.ToLower(cfg.Engine) {
	case "mysql":
		connString = "%s:%s@tcp(%s)/%s?%s"
	case "sqlserver":
		connString = "sqlserver://%s:%s@%s?database=%s&%s"
		if cfg.Domain != "" {
			usernameAndMaybeDomain = fmt.Sprintf("%s%%5C%s", cfg.Domain, cfg.Username)
		}
	case "sqlite":
		if cfg.Address != "" {
			return cfg.Address
		}
		return ":memory:"
	default:
		return ""
	}
	return fmt.Sprintf(
		connString,
		usernameAndMaybeDomain,
		cfg.Password,
		cfg.Address,
		cfg.DatabaseName,
		cfg.Options,
	)
}

func (cfg *Config) MigrationAddress() string {
	return fmt.Sprintf("%s://%s", cfg.Engine, cfg.FullAddress())
}

func (cfg *Config) loadFile() error {
	f, err := os.Open(cfg.ConfigLocation)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(data, &cfg.ConfigSecrets); err != nil {
		return err
	}

	return nil
}
