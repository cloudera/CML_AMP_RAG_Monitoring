package sbswagger

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-openapi/runtime"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
)

type ErrorResponder lhttp.HttpError

func (e ErrorResponder) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {
	var code int
	var message errorResponderMessage
	if e.Err != nil {
		code = http.StatusInternalServerError
		message.Message = "Internal server error"
	} else {
		code = e.Code
		message.Message = e.Message
	}

	rw.WriteHeader(code)
	if message.Message != "" {
		if err := producer.Produce(rw, message.Message); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

type errorResponderMessage struct {
	Message string

	counter int
}

func (m *errorResponderMessage) Read(p []byte) (n int, err error) {
	if m.counter >= len(m.Message) {
		return 0, io.EOF
	}

	msg := m.Message[m.counter:]
	msg = msg[:len(p)]
	m.counter += len(p)

	return copy(p, []byte(msg)), nil
}

func (m *errorResponderMessage) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Message)
}

var _ io.Reader = &errorResponderMessage{}
var _ json.Marshaler = &errorResponderMessage{}
