package interceptors

import (
	"bytes"
	"io"
	"sync"

	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

var logIOInterceptorPool = &sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

type LogIOConfig struct {
	Enabled bool
}

func HttpServerLogIOInterceptor(cfg *LogIOConfig) sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		if cfg.Enabled {
			bufferRead := logIOInterceptorPool.Get().(*bytes.Buffer)
			defer logIOInterceptorPool.Put(bufferRead)
			defer bufferRead.Reset()
			bufferWrite := logIOInterceptorPool.Get().(*bytes.Buffer)
			defer logIOInterceptorPool.Put(bufferWrite)
			defer bufferWrite.Reset()

			teeReader := io.TeeReader(request.Request.Body, bufferRead)
			multiWriter := io.MultiWriter(request.Writer, bufferWrite)

			newBody := &wrappers.Request{
				Original: request.Request.Body,
				Reader:   teeReader,
			}

			newW := wrappers.CustomizableResponseWriter{
				Response: request.Writer,
				Writer:   multiWriter,
			}

			next(request.WithBody(newBody).WithWriter(&newW))
		} else {
			next(request)
		}
	}
}
