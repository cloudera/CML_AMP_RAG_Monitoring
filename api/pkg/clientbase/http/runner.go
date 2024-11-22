package cbhttp

import (
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"io/ioutil"
	"net/http"
)

func httpDoNoRetry(client *http.Client, r *Request) (*Response, *lhttp.HttpError) {
	request, err := http.NewRequest(r.Method, r.URI, r.Body)
	if err != nil {
		return nil, &lhttp.HttpError{Err: err}
	}

	// Add headers
	if r.Header != nil {
		request.Header = r.Header
	}

	// Add query elements
	if r.Query != nil {
		request.URL.RawQuery = r.Query.Encode()
	}

	request.ContentLength = r.ContentLength

	if r.Context != nil {
		request = request.WithContext(r.Context)
	}

	// Make the request
	resp, err := client.Do(request)
	if err != nil {
		return nil, &lhttp.HttpError{Err: err}
	}

	if r.Body != nil {
		if err := r.Body.Close(); err != nil {
			return nil, &lhttp.HttpError{Err: err}
		}
	}

	if resp.StatusCode < 200 || 300 <= resp.StatusCode {
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, &lhttp.HttpError{Err: err}
		}
		return nil, &lhttp.HttpError{Code: resp.StatusCode, Message: string(responseBody)}
	}

	response := &Response{*resp}
	return response, nil
}
