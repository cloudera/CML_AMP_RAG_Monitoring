package interceptors

import (
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

func HttpServerDefaultContentTypeInterceptor(t string) sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		w := &wrappers.CustomizableResponseWriter{
			Response: request.Writer,
			OnWriteHeader: func(w *wrappers.CustomizableResponseWriter, code int) {
				if request.Writer.Header().Get("Content-Type") == "" {
					request.Writer.Header().Set("Content-Type", t)
				}
				request.Writer.WriteHeader(code)
			},
		}
		next(request.WithWriter(w))
	}
}
