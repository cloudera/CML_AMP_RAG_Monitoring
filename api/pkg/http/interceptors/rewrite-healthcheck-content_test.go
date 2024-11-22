package interceptors

import (
	"github.com/stretchr/testify/assert"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	sbhttptest "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/test"
	"net/http"
	"net/http/httptest"
	"pgregory.net/rapid"
	"testing"
)

func TestRewriteHealthcheckContentInterceptor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		interceptor := HttpServerRewriteHealthcheckContentInterceptor()

		recorder := httptest.NewRecorder()

		// This header always rewrites the output regardless of what the input is
		request := sbhttptest.RequestGenerator(recorder).Draw(t, "request")
		handler := outputHandler(t)
		interceptor(request, handler)

		expectedOutput := []byte("OK")
		actualOutput := recorder.Body.Bytes()
		assert.Equal(t, http.StatusOK, recorder.Code)
		assert.EqualValues(t, expectedOutput, actualOutput)
	})
}

func outputHandler(t *rapid.T) sbhttpbase.HandleFunc {
	return func(request *sbhttpbase.Request) {
		// Just return output like PyMS does
		ogOutput := "{\"version\": \"2.0\"}"
		_, err := request.Writer.Write([]byte(ogOutput))
		assert.Nil(t, err)
		request.Writer.WriteHeader(http.StatusOK)
	}
}
