package sbhttpserver

import (
	"github.com/dimfeld/httptreemux"
	log "github.com/sirupsen/logrus"
	"github.com/uber/jaeger-client-go/log/zap"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/interceptors"
	context_cancel "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/interceptors/context-cancel"
	"net/http"
	"strings"

	sbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

func (instance *Instance) registerHandlers(server Server) error {
	for _, handle := range server.GetHandlers() {
		if handle.NotFound {
			log.Printf("registering not found handler")
			instance.registerNotFoundHandler(server, handle)
		} else {
			log.Printf("registering handler %s %s", handle.Method, handle.Path)
			if err := instance.registerHandler(server, handle); err != nil {
				return err
			}
		}
	}

	return nil
}

func (instance *Instance) createTailMiddlewares(path, method string) ([]sbhttpbase.MiddlewareFunc, error) {
	return []sbhttpbase.MiddlewareFunc{
		interceptors.HttpServerDefaultContentTypeInterceptor("application/json").Register(path, method),
		exhaustRequest,
		defaultOk,
		context_cancel.Interceptor{}.ToHTTP(),
		interceptors.HttpServerRecoverInterceptor().Register(path, method),
		interceptors.HttpSetCookieMirrorInterceptor().Register(path, method),
	}, nil
}

func (instance *Instance) registerHandler(server Server, handle HandleDescription) error {
	tail, err := instance.createTailMiddlewares(handle.Path, handle.Method)
	if err != nil {
		return err
	}

	middleware := make([]sbhttpbase.MiddlewareFunc, 0)
	for _, m := range handle.Middleware {
		middleware = append(middleware, m.Register(handle.Path, handle.Method))
	}

	middleware = append(middleware, tail...)

	handler := handle.Handler
	if len(middleware) > 0 {
		handler = ComposeMiddleware(middleware, handle.Handler)
	}

	instance.RegisterHandler(&HandleDescription{
		NotFound: handle.NotFound,
		Path:     handle.Path,
		Method:   handle.Method,
		Handler:  handler,
	})

	return nil
}

type notFoundWrapper struct {
	instance *Instance
	handle   HandleDescription
}

// TODO: remove this as we don't need a special handler
func (wrap *notFoundWrapper) Handle(request *sbhttpbase.Request) {
	h := wrap.handle
	h.Path = request.Request.URL.Path
	h.Method = request.Request.Method

	tail, err := wrap.instance.createTailMiddlewares(h.Path, h.Method)
	if err != nil {
		sbhttp.ReturnError(request.Writer, http.StatusInternalServerError, "failed to register middlewares", err)
		return
	}

	middleware := make([]sbhttpbase.MiddlewareFunc, 0)
	for _, m := range h.Middleware {
		middleware = append(middleware, m.Register(h.Path, h.Method))
	}
	middleware = append(middleware, tail...)

	handler := h.Handler
	if h.Middleware != nil {
		handler = ComposeMiddleware(middleware, h.Handler)
	}

	handler(request)
}

func (instance *Instance) registerNotFoundHandler(server Server, handle HandleDescription) {
	wrapper := &notFoundWrapper{
		instance: instance,
		handle:   handle,
	}

	instance.RegisterHandler(&HandleDescription{
		NotFound: handle.NotFound,
		Path:     handle.Path,
		Method:   handle.Method,
		Handler:  wrapper.Handle,
	})
}

type notFoundHandler struct {
	handler sbhttpbase.HandleFunc
	logger  *zap.Logger
}

func (h *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler(&sbhttpbase.Request{
		PathPattern: "*",
		Writer:      w,
		Request:     r,
	})
}

func handleWrapper(pathPattern string, handler sbhttpbase.HandleFunc) httptreemux.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		handler(&sbhttpbase.Request{
			PathPattern: pathPattern,
			Writer:      w,
			Request:     r,
			Params:      params,
		})
	}
}

func (b *Instance) RegisterHandler(handle *HandleDescription) {
	if handle.NotFound {
		b.router.NotFoundHandler = (&notFoundHandler{
			handler: handle.Handler,
		}).ServeHTTP
		return
	}

	switch handle.Method {
	case "*":
		for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
			b.RegisterHandler(&HandleDescription{
				Path:    handle.Path,
				Method:  method,
				Handler: handle.Handler,
			})
		}
	default:
		b.router.Handle(handle.Method, handle.Path, handleWrapper(handle.Path, handle.Handler))
	}

	if handle.Path[len(handle.Path)-1] != '/' && handle.Method != "*" && !strings.Contains(handle.Path, "*") {
		b.RegisterHandler(&HandleDescription{
			Method:  handle.Method,
			Path:    handle.Path + "/",
			Handler: handle.Handler,
		})
	}
}

func ComposeMiddleware(funcs []sbhttpbase.MiddlewareFunc, base sbhttpbase.HandleFunc) sbhttpbase.HandleFunc {
	for i := len(funcs) - 1; i >= 0; i-- {
		f := funcs[i]
		if f == nil {
			continue
		}
		oldBase := base
		base = func(request *sbhttpbase.Request) {
			f(request, oldBase)
		}
	}

	return base
}
