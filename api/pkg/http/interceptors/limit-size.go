package interceptors

import (
	"fmt"
	"io"
	"k8s.io/apimachinery/pkg/api/resource"
	"net/http"
	"strconv"

	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

func HttpServerLimitSizeInterceptor(size resource.Quantity) sbhttpbase.MiddlewareFunc {
	if size.Value() == 0 {
		return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
			next(request)
		}
	}

	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		maxLength := request.Request.Header.Get("Content-length")
		if maxLength == "" {
			sbhttp.ReturnError(request.Writer, http.StatusLengthRequired, "Content length header is empty", nil)
			return
		}

		length, err := strconv.Atoi(maxLength)
		if err != nil {
			sbhttp.ReturnError(request.Writer, http.StatusBadRequest, "Failed to convert content length header to integer", err)
			return
		}

		if int64(length) > size.Value() {
			sbhttp.ReturnError(request.Writer, http.StatusRequestEntityTooLarge, fmt.Sprintf("Content length header is bigger than %d bytes", size.Value()), nil)
			return
		}

		next(request.WithBody(&wrappers.Request{
			Original: request.Request.Body,
			Reader:   io.LimitReader(request.Request.Body, int64(length)),
		}))
	}
}
