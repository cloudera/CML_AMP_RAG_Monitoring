package cbhttp

import lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"

type Client interface {
	Do(r *Request, m ...MiddlewareFunc) (*Response, *lhttp.HttpError)
}
