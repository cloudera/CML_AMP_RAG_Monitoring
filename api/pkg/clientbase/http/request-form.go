package cbhttp

import (
	"bytes"
	"fmt"
	"mime/multipart"

	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
)

func FormFields(fields map[string]string) RequestOption {
	return func(r *Request) *Request {
		requestBody := &bytes.Buffer{}
		writer := multipart.NewWriter(requestBody)
		// Force a boundary since the python side doesn't understand not having boundaries
		boundary := "---------------------"
		writer.SetBoundary(boundary)

		// Fill in the form fields
		for key, val := range fields {
			err := writer.WriteField(key, val)
			if err != nil {
				r.HErr = &lhttp.HttpError{Err: err}
				return r
			}
		}

		// Close the form body
		err := writer.Close()
		if err != nil {
			r.HErr = &lhttp.HttpError{Err: err}
			return r
		}

		return r.Options(Body(requestBody), AddHeader("content-type", fmt.Sprintf("multipart/form-data; boundary=%s", boundary)))
	}
}
