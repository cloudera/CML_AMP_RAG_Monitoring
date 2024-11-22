package cbhttp

import (
	"bytes"
	"github.com/avast/retry-go"
	"io"
	"io/ioutil"
	"net/http"

	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
)

func (c *Instance) composeMiddleware(funcs []MiddlewareFunc, runner RunnerFunc) RunnerFunc {
	if runner == nil {
		runner = c.do
	}

	if len(funcs) == 0 {
		return runner
	}
	if funcs[0] == nil {
		return c.composeMiddleware(funcs[1:], runner)
	}
	return funcs[0](c.composeMiddleware(funcs[1:], runner))
}

func (c *Instance) With(newMiddlewares ...MiddlewareFunc) *Instance {
	return &Instance{
		Client: c.Client,
		runner: c.composeMiddleware(newMiddlewares, c.runner),
	}
}

// Do sends an HTTP request and returns an HTTP response.
//
// An error is returned for network/policy issues as well as for non-2xx responses. This differs from the standard
// library's http.Client.Do, which does not return an error for the latter.
func (c *Instance) Do(r *Request, m ...MiddlewareFunc) (*Response, *lhttp.HttpError) {
	if r.HErr != nil {
		return nil, r.HErr
	}

	if len(m) > 0 {
		return c.With(m...).Do(r)
	}

	runner := c.runner
	if runner == nil {
		runner = c.do
	}
	return runner(r)
}

func (c *Instance) DoNoResponse(r *Request, m ...MiddlewareFunc) *lhttp.HttpError {
	body, err := c.Do(r, m...)
	if body != nil {
		body.Close()
	}
	return err
}

func (c *Instance) do(r *Request) (*Response, *lhttp.HttpError) {
	if len(r.retryOptions) > 0 {
		opts := append(r.retryOptions, retry.Context(r.Context))

		var response *Response
		var herr *lhttp.HttpError

		var bodyContent []byte
		var err error
		if r.Body != nil {
			// We have to keep a copy of the body in case of retries
			bodyContent, err = ioutil.ReadAll(r.Body)
			if err != nil {
				return nil, lhttp.FromError(err)
			}
			r.Body.Close()
		}

		_ = retry.Do(func() error {
			if r.Body != nil {
				r.Body = ioutil.NopCloser(bytes.NewBuffer(bodyContent))
			}
			response, herr = c.doNoRetry(r)
			return herr
		}, opts...)

		return response, herr
	} else {
		return c.doNoRetry(r)
	}
}

func (c *Instance) Close() error {
	if c.Client != nil {
		c.Client.CloseIdleConnections()
	}
	return nil
}

type Response struct {
	http.Response
}

func (r *Response) Read(p []byte) (int, error) { return r.Body.Read(p) }
func (r *Response) Close() error               { return r.Body.Close() }

var _ io.ReadCloser = &Response{}
