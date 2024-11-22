package cbhttp

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"

	lgzip "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/gzip"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func BodyObj(obj interface{}) RequestOption {
	return func(r *Request) *Request {
		buffer := &bytes.Buffer{}

		if protoObj, ok := obj.(proto.Message); ok {
			if content, err := protojson.Marshal(protoObj); err != nil {
				r.HErr = &lhttp.HttpError{Err: err}
				return r
			} else {
				buffer = bytes.NewBuffer(content)
			}
		} else {
			if err := json.NewEncoder(buffer).Encode(obj); err != nil {
				r.HErr = &lhttp.HttpError{Err: err}
				return r
			}
		}

		r.Body = ioutil.NopCloser(buffer)
		return AddHeader("content-type", "application/json")(r)
	}
}

func Body(reader io.Reader) RequestOption {
	if readcloser, ok := reader.(io.ReadCloser); ok {
		return func(r *Request) *Request {
			r.Body = readcloser
			return r
		}
	} else {
		return func(r *Request) *Request {
			r.Body = ioutil.NopCloser(reader)
			return r
		}
	}
}

func GzipBody(reader io.Reader) RequestOption {
	zipped, err := lgzip.NewCompressReader(reader)
	if err != nil {
		return func(r *Request) *Request {
			r.HErr.Err = err
			return r
		}
	}

	return func(r *Request) *Request {
		return Body(zipped)(AddHeader("Content-Encoding", "gzip")(r))
	}
}
