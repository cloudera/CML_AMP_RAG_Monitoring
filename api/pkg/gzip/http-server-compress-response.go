package lgzip

import (
	"bufio"
	"compress/gzip"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"io/ioutil"
	"strings"
	"sync"
)

var gzipWriterPool = &sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(ioutil.Discard)
	},
}

var bufferedWriterPool = &sync.Pool{
	New: func() interface{} {
		return bufio.NewWriter(ioutil.Discard)
	},
}

func HttpServerCompressResponseInterceptor() sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		encodings := request.Request.Header.Values("Accept-Encoding")
		acceptsGzip := false
		for _, encoding := range encodings {
			for _, localEncoding := range strings.Split(encoding, ",") {
				if strings.EqualFold(strings.Trim(localEncoding, " "), "gzip") {
					acceptsGzip = true
					break
				}
			}
		}

		if !acceptsGzip {
			next(request)
			return
		}

		// Don't pass the information downstream to avoid double encoding
		request.Request.Header.Del("Accept-Encoding")

		bufferedWriter := bufferedWriterPool.Get().(*bufio.Writer)
		defer bufferedWriterPool.Put(bufferedWriter)
		bufferedWriter.Reset(request.Writer)
		defer bufferedWriter.Flush()

		gzipWriter := gzipWriterPool.Get().(*gzip.Writer)
		defer gzipWriterPool.Put(gzipWriter)
		gzipWriter.Reset(bufferedWriter)
		defer gzipWriter.Close()
		request.Writer.Header().Set("content-encoding", "gzip")

		next(request.WithWriter(&wrappers.CustomizableResponseWriter{
			Response: request.Writer,
			Writer:   gzipWriter,
		}))
	}
}
