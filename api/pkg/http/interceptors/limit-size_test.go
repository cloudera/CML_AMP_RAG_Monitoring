package interceptors

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	sbhttptest "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/test"
	"io"
	"k8s.io/apimachinery/pkg/api/resource"
	"net/http"
	"net/http/httptest"
	"pgregory.net/rapid"
	"testing"
)

func TestLimitSizeInterceptor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxSize := rapid.Int64Range(0, 20).Draw(t, "maxSize")
		qty := resource.Quantity{}
		qty.Set(maxSize)
		interceptor := HttpServerLimitSizeInterceptor(qty)

		recorder := httptest.NewRecorder()
		body := rapid.SliceOfN(rapid.Byte(), 0, 30).Draw(t, "body")
		bodyLen := len(body)
		request := sbhttptest.RequestWithBodyGenerator(recorder, bytes.NewReader(body)).Draw(t, "request")
		request.Request.Header.Set("Content-length", fmt.Sprintf("%d", bodyLen))
		handler := handler(t, body)
		interceptor(request, handler)
		// The interceptor considers 0 to mean "no limit"
		if maxSize != 0 && bodyLen > int(maxSize) {
			assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
		} else {
			assert.Equal(t, http.StatusOK, recorder.Code)
		}
	})
}

func handler(t *rapid.T, expectedBody []byte) sbhttpbase.HandleFunc {
	return func(request *sbhttpbase.Request) {
		// Just assert the body is the same
		var actualBodyBuffer bytes.Buffer
		_, err := io.Copy(&actualBodyBuffer, request.Request.Body)
		assert.Nil(t, err)
		assert.Nil(t, request.Request.Body.Close())
		actualBody := actualBodyBuffer.Bytes()

		// TODO: Figure out how to compare empty and nil slices
		// Having issues with
		//     Not equal:
		//     expected: []byte{}
		//     actual  : []byte(nil)
		// So I'm just going to compare length of empty body for now
		if len(expectedBody) == 0 {
			assert.Equal(t, len(expectedBody), len(actualBody))
		} else {
			assert.Equal(t, expectedBody, actualBody)
		}

		// Output is not affected by this handler
	}
}
