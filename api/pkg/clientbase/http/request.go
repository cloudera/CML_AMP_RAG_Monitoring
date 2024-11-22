package cbhttp

import (
	"context"
	"io"
	"net/http"
	"net/url"

	retry "github.com/avast/retry-go"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
)

type Request struct {
	Method        string
	URI           string
	Header        http.Header
	Query         url.Values
	Body          io.ReadCloser
	ContentLength int64
	HErr          *lhttp.HttpError
	Context       context.Context
	retryOptions  []retry.Option
}

type RequestOption func(*Request) *Request

func NewRequest(ctx context.Context, method, uri string, options ...RequestOption) *Request {
	r := &Request{
		Method:  method,
		URI:     uri,
		Context: ctx,
	}

	return r.Options(options...)
}

func (r *Request) Options(options ...RequestOption) *Request {
	return ComposeOptions(options...)(r)
}

func (r *Request) Clone() *Request {
	var newHeader http.Header
	if r.Header != nil {
		newHeader = http.Header.Clone(r.Header)
	}
	var newQuery url.Values
	if r.Query != nil {
		newQuery = url.Values(http.Header.Clone(http.Header(r.Query)))
	}

	return &Request{
		Method:        r.Method,
		URI:           r.URI,
		Header:        newHeader,
		Query:         newQuery,
		Body:          r.Body,
		ContentLength: r.ContentLength,
		HErr:          r.HErr.Clone(),
		Context:       r.Context,
	}
}

func ComposeOptions(options ...RequestOption) RequestOption {
	return func(r *Request) *Request {
		for _, opt := range options {
			r = opt(r)
		}
		return r
	}
}
