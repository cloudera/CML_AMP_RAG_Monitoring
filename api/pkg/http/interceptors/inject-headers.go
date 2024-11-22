package interceptors

import (
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"net/http"
	"strings"
)

func InjectHeadersInterceptor(headers http.Header) sbhttpbase.MiddlewareFunc {
	for k, v := range headers {
		if strings.Contains(k, "_") {
			delete(headers, k)
			headers[strings.ReplaceAll(k, "_", "-")] = v
		}
	}

	wrapper := wrappers.CustomizableResponseWriter{
		OnWriteHeader: func(w *wrappers.CustomizableResponseWriter, code int) {
			for k, vals := range headers {
				w.Response.Header().Del(k)
				for _, v := range vals {
					w.Response.Header().Add(k, v)
				}
			}
			w.Response.WriteHeader(code)
		},
	}

	return wrapper.AsInterceptor()
}
