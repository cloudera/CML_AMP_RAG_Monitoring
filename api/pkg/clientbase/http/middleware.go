package cbhttp

import (
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
)

type RunnerFunc func(r *Request) (*Response, *lhttp.HttpError)
type MiddlewareFunc func(next RunnerFunc) RunnerFunc
