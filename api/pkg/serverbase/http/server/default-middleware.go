package sbhttpserver

import (
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"io"
	"io/ioutil"
)

func exhaustRequest(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
	next(request)

	// Exhaust the request so that it can be re-used
	io.Copy(ioutil.Discard, request.Request.Body)
	request.Request.Body.Close()
}

func defaultOk(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
	response := wrappers.CustomizableResponseWriter{
		Response: request.Writer,
		OnWrite: func(w *wrappers.CustomizableResponseWriter, p []byte) (int, error) {
			if w.Code == 0 {
				w.WriteHeader(200)
			}
			return w.Response.Write(p)
		},
	}

	next(request.WithWriter(&response))

	if response.Code == 0 {
		request.Writer.WriteHeader(200)
	}
}
