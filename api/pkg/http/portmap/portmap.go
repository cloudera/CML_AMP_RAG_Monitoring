package portmap

import (
	"fmt"
	"github.com/spf13/afero"
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
	v1 "k8s.io/api/core/v1"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
)

type HostPort struct {
	Host       string
	PortName   string
	PortNumber int64
}

func (h *HostPort) String() string {
	if h.PortNumber != 0 {
		return fmt.Sprintf("%s:%d", h.Host, h.PortNumber)
	}
	return h.Host
}

type Config struct {
	MapFile string `env:"PORTMAP_FILE_LOCATION"`
	mapping []v1.ContainerPort
}

func NewConfigFromEnv() (*Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return nil, err
	}
	if cfg.MapFile != "" {
		err = lconfig.LoadStaticYamlConfig(cfg.MapFile, afero.NewOsFs(), &cfg.mapping)
		if err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}

func (c *Config) Mapping() []v1.ContainerPort {
	return c.mapping
}

func (c *Config) parseHostPort(v string) (HostPort, error) {
	ret := HostPort{}

	host, port, err := net.SplitHostPort(v)
	if err != nil {
		return ret, err
	}

	ret.Host = host
	ret.PortName = port

	match, err := regexp.MatchString(`^\d+$`, port)
	if err != nil {
		return ret, err
	}
	if match {
		portNumber, err := strconv.ParseInt(port, 10, 32)
		if err != nil {
			return ret, err
		}
		ret.PortNumber = portNumber
		return ret, nil
	}

	for _, p := range c.mapping {
		if p.Name == port {
			ret.PortNumber = int64(p.ContainerPort)
			return ret, nil
		}
	}

	return ret, fmt.Errorf("failed to find port for name \"%s\"", port)
}

func (c *Config) UrlParseFunc() lconfig.ParseFuncs {
	ret := make(lconfig.ParseFuncs)
	ret[reflect.TypeOf(&url.URL{})] = func(v string) (interface{}, error) {
		url, err := url.Parse(v)
		if err != nil {
			return nil, err
		}

		hostPort, err := c.parseHostPort(url.Host)
		if err != nil {
			return nil, err
		}

		url.Host = hostPort.String()
		return url, nil
	}
	ret[reflect.TypeOf(HostPort{})] = func(v string) (interface{}, error) {
		return c.parseHostPort(v)
	}

	return ret
}
