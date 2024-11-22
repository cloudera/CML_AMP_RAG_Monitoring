package interceptors

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"sync"
)

func HttpServerRewriteHealthcheckContentInterceptor() sbhttpbase.MiddlewareFunc {
	/*
		This interceptor is SPECIFICALLY designed for rewriting `{"version": "2.0"}` into something more palatable
		for the user.
	*/

	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		newContent := "OK"
		newBytes := []byte(newContent)
		responseBuf := dataLoggingPool.Get().(*bytes.Buffer)
		newWriter := writerPool.Get().(*wrappers.CustomizableResponseWriter)
		defer writerPool.Put(newWriter)
		newWriter.Response = request.Writer
		newWriter.Writer = responseBuf

		next(request.WithWriter(newWriter))

		_, err := request.Writer.Write(newBytes)
		if err != nil {
			log.Printf("failed to write response body: %s", err)
		}

	}
}

var dataLoggingPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

var writerPool = sync.Pool{
	New: func() interface{} {
		return &wrappers.CustomizableResponseWriter{
			OnWriteHeader: func(w *wrappers.CustomizableResponseWriter, code int) {
				// Do nothing so that we don't send the code upstream until we're done
			},
		}
	},
}
