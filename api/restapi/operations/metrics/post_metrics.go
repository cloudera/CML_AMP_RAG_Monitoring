// Code generated by go-swagger; DO NOT EDIT.

package metrics

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PostMetricsHandlerFunc turns a function with the right signature into a post metrics handler
type PostMetricsHandlerFunc func(PostMetricsParams) middleware.Responder

// Handle executing the request and returning a response
func (fn PostMetricsHandlerFunc) Handle(params PostMetricsParams) middleware.Responder {
	return fn(params)
}

// PostMetricsHandler interface for that can handle valid post metrics params
type PostMetricsHandler interface {
	Handle(PostMetricsParams) middleware.Responder
}

// NewPostMetrics creates a new http.Handler for the post metrics operation
func NewPostMetrics(ctx *middleware.Context, handler PostMetricsHandler) *PostMetrics {
	return &PostMetrics{Context: ctx, Handler: handler}
}

/*
	PostMetrics swagger:route POST /metrics metrics postMetrics

Create metrics.

Create monitoring metrics
*/
type PostMetrics struct {
	Context *middleware.Context
	Handler PostMetricsHandler
}

func (o *PostMetrics) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPostMetricsParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
