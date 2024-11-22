package swagger

import (
	"github.com/go-openapi/runtime"
)

type RichError struct {
	originalError error

	id       string
	method   string
	path     string
	params   string
	code     int
	response string
}

func NewRichError(operation *runtime.ClientOperation, err error) (*RichError, error) {
	ret := &RichError{
		originalError: err,
		id:            operation.ID,
		method:        operation.Method,
		path:          operation.PathPattern,
	}

	if response, ok := err.(SwaggerResponse); ok {
		ret.code = response.Code()
		payload, perr := response.GetSerializedPayload()
		if perr != nil {
			return nil, perr
		}
		ret.response = string(payload)
	}

	if response, ok := err.(*runtime.APIError); ok {
		ret.code = response.Code
		if payload, pok := response.Response.(runtime.ClientResponse); pok {
			ret.response = payload.Message()
		}
	}

	if params, ok := operation.Params.(SwaggerParams); ok {
		data, derr := params.GetSerializedParams()
		if derr != nil {
			return nil, derr
		}
		ret.params = string(data)
	}

	return ret, nil
}

func (e *RichError) Error() string {
	return e.originalError.Error()
}

func (e *RichError) ID() string       { return e.id }
func (e *RichError) Method() string   { return e.method }
func (e *RichError) Path() string     { return e.path }
func (e *RichError) Params() string   { return e.params }
func (e *RichError) Code() int        { return e.code }
func (e *RichError) Response() string { return e.response }
