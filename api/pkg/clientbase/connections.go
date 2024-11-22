package clientbase

import (
	cbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase/http"
)

type Connections struct {
	Cfg        *Config
	HttpClient *cbhttp.Instance
}

func NewConnections(cfg *Config, httpClient *cbhttp.Instance) (*Connections, error) {
	c := &Connections{
		Cfg: cfg,
	}

	c.HttpClient = httpClient

	return c, nil
}

func (c *Connections) Close() {

}
