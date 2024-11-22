package cbhttp

import (
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"net/http"
)

type Instance struct {
	Client    *http.Client
	runner    RunnerFunc
	doNoRetry RunnerFunc
}

var _ Client = &Instance{}

func NewInstance(cfg *Config) (*Instance, error) {
	var checkRedirect func(req *http.Request, via []*http.Request) error

	if cfg.AvoidRedirects {
		checkRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	client := &http.Client{
		Timeout:       cfg.Timeout,
		CheckRedirect: checkRedirect,
	}

	return &Instance{
		Client: client,
		doNoRetry: func(r *Request) (*Response, *lhttp.HttpError) {
			return httpDoNoRetry(client, r)
		},
	}, nil
}
