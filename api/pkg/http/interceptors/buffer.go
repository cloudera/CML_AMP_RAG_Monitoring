package interceptors

import (
	"bytes"
	"io"
	"net/http"
	"sync"

	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

var bufferInterceptorPool = &sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func HttpServerBufferInterceptor(enabled bool) sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		if enabled {
			bufferRead := bufferInterceptorPool.Get().(*bytes.Buffer)
			defer bufferInterceptorPool.Put(bufferRead)
			defer bufferRead.Reset()
			bufferWrite := bufferInterceptorPool.Get().(*bytes.Buffer)
			defer bufferInterceptorPool.Put(bufferWrite)
			defer bufferWrite.Reset()

			if _, err := io.Copy(bufferRead, request.Request.Body); err != nil {
				request.Writer.WriteHeader(http.StatusInternalServerError)
				return
			}

			newBody := &wrappers.Request{
				Original: request.Request.Body,
				Reader:   bufferRead,
			}

			newW := wrappers.CustomizableResponseWriter{
				Response: request.Writer,
				Writer:   bufferWrite,
			}

			next(request.WithBody(newBody).WithWriter(&newW))

			if _, err := io.Copy(request.Writer, bufferWrite); err != nil {
				return
			}
		} else {
			next(request)
		}
	}
}
