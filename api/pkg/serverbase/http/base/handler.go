package sbhttpbase

import (
	"net/http"
)

type HandleFunc func(request *Request)

func HandleStdFunc(fn func(w http.ResponseWriter, r *http.Request)) HandleFunc {
	return func(request *Request) {
		fn(request.Writer, request.Request)
	}
}

type HandleInfo struct {
	NotFound bool
	Path     string
	Method   string
}

// Internal version of the interface that uses a pre-define logger
// We use it for the internal handlers as they can use the logger from the base
type HandleDescription struct {
	NotFound bool
	Path     string
	Method   string
	Handler  HandleFunc
}
