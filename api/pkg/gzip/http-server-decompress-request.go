package lgzip

import (
	"bufio"
	"compress/gzip"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"net/http"
	"sync"
)

var gzipReaderPool = &sync.Pool{
	New: func() interface{} {
		return &gzip.Reader{}
	},
}

var bufferedReaderPool = &sync.Pool{
	New: func() interface{} {
		return bufio.NewReader(nil)
	},
}

func HttpServerDecompressRequestInterceptor() sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		if request.Request.Header.Get("Content-Encoding") != "gzip" {
			next(request)
			return
		}

		bufferedReader := bufferedReaderPool.Get().(*bufio.Reader)
		defer bufferedReaderPool.Put(bufferedReader)
		bufferedReader.Reset(request.Request.Body)

		gzipReader := gzipReaderPool.Get().(*gzip.Reader)
		defer gzipReaderPool.Put(gzipReader)
		if err := gzipReader.Reset(bufferedReader); err != nil {
			sbhttp.ReturnError(request.Writer, http.StatusBadRequest, "failed to decompress request", err)
			return
		}

		next(request.WithBody(&wrappers.Request{
			Original: request.Request.Body,
			Reader:   gzipReader,
		}))
	}
}
