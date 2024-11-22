package lhttp

import (
	"fmt"
	lswagger "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/swagger"
	"net/http"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/go-openapi/runtime"
	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/runtime"
	"google.golang.org/grpc/status"
)

type HttpError struct {
	Code    int
	Message string
	Err     error
}

func FromError(err error) *HttpError {
	if err == nil {
		return nil
	}

	// gRPC
	if st, ok := status.FromError(err); ok {
		return &HttpError{
			Code:    grpcruntime.HTTPStatusFromCode(st.Code()),
			Message: st.Message(),
		}
	}

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case s3.ErrCodeNoSuchBucket:
			fallthrough
		case s3.ErrCodeNoSuchKey:
			return &HttpError{
				Code:    http.StatusNotFound,
				Message: aerr.Error(),
			}
		}
	}

	if richErr, ok := err.(*lswagger.RichError); ok {
		return &HttpError{
			Code:    richErr.Code(),
			Message: richErr.Response(),
		}
	}

	// Own type
	if herr, ok := err.(*HttpError); ok {
		return herr
	}

	return &HttpError{Err: err}
}

func (e *HttpError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("got code %d and message \"%s\"", e.Code, e.Message)
}

func (e *HttpError) Clone() *HttpError {
	return &HttpError{
		Code:    e.Code,
		Message: e.Message,
		Err:     e.Err,
	}
}

type responseMessage string

func (m responseMessage) MarshalBinary() ([]byte, error) {
	return []byte(m), nil
}

func (e *HttpError) WriteResponse(w http.ResponseWriter, producer runtime.Producer) {
	if e.Err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(e.Code)
		if e.Message != "" {
			var m responseMessage
			m = responseMessage(e.Message)
			if err := producer.Produce(w, m); err != nil {
				panic(err) // let the recovery middleware deal with this
			}
		}
	}
}

func (e *HttpError) WithPayload(payload string) *HttpError {
	e.Message = payload
	return e
}

func (e *HttpError) SetPayload(payload string) {
	e.Message = payload
}

func NewNotFound(message string) *HttpError {
	return &HttpError{Code: http.StatusNotFound, Message: message}
}

func NewBadGateway(message string) *HttpError {
	return &HttpError{Code: http.StatusBadGateway, Message: message}
}

func NewConflict(message string) *HttpError {
	return &HttpError{Code: http.StatusConflict, Message: message}
}

func NewBadRequest(message string) *HttpError {
	return &HttpError{Code: http.StatusBadRequest, Message: message}
}

func NewInternalError(message string) *HttpError {
	return &HttpError{Code: http.StatusInternalServerError, Message: message}
}

func NewForbidden() *HttpError {
	return &HttpError{Code: http.StatusForbidden, Message: "Forbidden"}
}
