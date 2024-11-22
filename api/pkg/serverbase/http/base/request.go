package sbhttpbase

import (
	"context"
	"io"
	"net/http"
)

type Request struct {
	PathPattern string
	Writer      http.ResponseWriter
	Request     *http.Request
	Params      map[string]string
}

func (r *Request) WithWriter(w http.ResponseWriter) *Request {
	newRequest := *r
	newRequest.Writer = w
	return &newRequest
}

func (r *Request) WithRequest(req *http.Request) *Request {
	newRequest := *r
	newRequest.Request = req
	return &newRequest
}

func (r *Request) WithContext(ctx context.Context) *Request {
	newRequest := *r
	newRequest.Request = r.Request.WithContext(ctx)
	return &newRequest
}

func (r *Request) WithBody(body io.ReadCloser) *Request {
	newRequest := *r
	reqCopy := *r.Request
	reqCopy.Body = body
	newRequest.Request = &reqCopy
	return &newRequest
}
