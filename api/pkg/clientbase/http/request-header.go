package cbhttp

import (
	"net/http"
)

func AddHeader(key, value string) RequestOption {
	return func(r *Request) *Request {
		if r.Header == nil {
			r.Header = make(map[string][]string)
		}
		r.Header.Add(key, value)
		return r
	}
}

func SetHeader(key, value string) RequestOption {
	return func(r *Request) *Request {
		if r.Header == nil {
			r.Header = make(map[string][]string)
		}
		r.Header.Set(key, value)
		return r
	}
}

func Header(h http.Header) RequestOption {
	return func(r *Request) *Request {
		if h != nil {
			r.Header = http.Header.Clone(h)
		}
		return r
	}
}
