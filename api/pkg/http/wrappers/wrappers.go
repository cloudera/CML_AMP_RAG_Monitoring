package wrappers

import (
	"fmt"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"io"
	"net/http"
)

type Request struct {
	Original io.ReadCloser
	Reader   io.Reader
}

func (r *Request) Read(p []byte) (int, error) {
	return r.Reader.Read(p)
}

func (r *Request) Close() error {
	if err := r.Original.Close(); err != nil {
		return err
	}
	if reader, ok := r.Reader.(io.ReadCloser); ok {
		return reader.Close()
	}
	return nil
}

type CustomizableResponseWriter struct {
	Response      http.ResponseWriter
	Writer        io.Writer
	Code          int
	OnHeader      func(w *CustomizableResponseWriter) http.Header
	OnWriteHeader func(w *CustomizableResponseWriter, code int)
	OnWrite       func(w *CustomizableResponseWriter, p []byte) (int, error)
}

func (w *CustomizableResponseWriter) Header() http.Header {
	if w.OnHeader != nil {
		return w.OnHeader(w)
	}
	if w.Response != nil {
		return w.Response.Header()
	}
	return nil
}

func (w *CustomizableResponseWriter) Write(p []byte) (int, error) {
	if w.OnWrite != nil {
		return w.OnWrite(w, p)
	}
	if w.Writer != nil {
		return w.Writer.Write(p)
	}
	if w.Response != nil {
		return w.Response.Write(p)
	}
	return 0, fmt.Errorf("no writer defined")
}

func (w *CustomizableResponseWriter) WriteHeader(code int) {
	w.Code = code
	if w.OnWriteHeader != nil {
		w.OnWriteHeader(w, code)
		return
	}
	if w.Response != nil {
		w.Response.WriteHeader(code)
	}
}

func (w *CustomizableResponseWriter) AsInterceptor() sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		local := *w
		if local.Response == nil {
			local.Response = request.Writer
		}
		if local.Writer == nil {
			local.Writer = request.Writer
		}

		next(request.WithWriter(&local))
	}
}
