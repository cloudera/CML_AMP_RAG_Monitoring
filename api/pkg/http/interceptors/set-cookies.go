package interceptors

import (
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

func HttpSetCookieMirrorInterceptor() sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		setCookies := request.Request.Header.Get("set-cookie")
		request.Writer.Header().Set("set-cookie", setCookies)
		next(request)
	}
}
