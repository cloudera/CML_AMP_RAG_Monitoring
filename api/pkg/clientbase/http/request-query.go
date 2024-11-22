package cbhttp

import (
	"net/url"

	"github.com/gorilla/schema"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
)

var schemaEncoder = schema.NewEncoder()

func init() {
	schemaEncoder.SetAliasTag("json")
}

func QueryObj(obj interface{}) RequestOption {
	return func(r *Request) *Request {
		query := url.Values{}
		if err := schemaEncoder.Encode(obj, query); err != nil {
			r.HErr = &lhttp.HttpError{Err: err}
		} else {
			r.Query = query
		}
		return r
	}
}
func Query(obj url.Values) RequestOption {
	return func(r *Request) *Request {
		r.Query = obj
		return r
	}
}
