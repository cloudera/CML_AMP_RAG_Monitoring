package interceptors

import (
	lhttptest "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/test"
	sbhttptest "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/test"
	"net/http/httptest"
	"pgregory.net/rapid"
	"testing"
)

func TestInjectHeadersInterceptor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		headers := lhttptest.HeadersGenerator().Draw(t, "header")
		interceptor := InjectHeadersInterceptor(headers)

		recorder := httptest.NewRecorder()
		request := sbhttptest.RequestGenerator(recorder).Draw(t, "request")
		handler := sbhttptest.HandlerGenerator().Draw(t, "handler")

		interceptor(request, handler)

		lhttptest.CheckHeaders(t, headers, request.Writer.Header())
	})
}
