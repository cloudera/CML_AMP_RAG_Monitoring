package cbhttp

import (
	"github.com/avast/retry-go"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"time"
)

func RetryMaxDelay(dt time.Duration) RequestOption {
	return func(r *Request) *Request {
		r.retryOptions = append(r.retryOptions, retry.MaxDelay(dt))
		return r
	}
}

type RetryIfFunc func(httpError *lhttp.HttpError) bool

func RetryIf(fn RetryIfFunc) RequestOption {
	return func(r *Request) *Request {
		r.retryOptions = append(r.retryOptions, retry.RetryIf(func(err error) bool {
			return fn(err.(*lhttp.HttpError))
		}))
		return r
	}
}

func RetryIfBaseError(httpError *lhttp.HttpError) bool {
	if httpError == nil {
		return false
	}
	return httpError.Err != nil
}

type OnRetryFunc func(n uint, err *lhttp.HttpError)

func OnRetry(fn OnRetryFunc) RequestOption {
	return func(r *Request) *Request {
		r.retryOptions = append(r.retryOptions, retry.OnRetry(func(n uint, err error) {
			fn(n, err.(*lhttp.HttpError))
		}))
		return r
	}
}

func RetryAttempts(attempts uint) RequestOption {
	return func(r *Request) *Request {
		r.retryOptions = append(r.retryOptions, retry.Attempts(attempts))
		return r
	}
}

func RetryFixedDelay(d time.Duration) RequestOption {
	return func(r *Request) *Request {
		r.retryOptions = append(r.retryOptions, retry.Delay(d), retry.DelayType(retry.FixedDelay))
		return r
	}
}
