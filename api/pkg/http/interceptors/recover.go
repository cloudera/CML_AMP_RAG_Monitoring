package interceptors

import (
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

func HttpServerRecoverInterceptor() sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		defer func() {
			if r := recover(); r != nil {
				request.Writer.WriteHeader(500)
				request.Writer.Write([]byte("Internal server error"))
			}
		}()
		next(request)
	}
}
