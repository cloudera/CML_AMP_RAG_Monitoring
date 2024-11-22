package cbhttpmiddleware

import (
	"bytes"
	"encoding/json"
	"io"

	cbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase/http"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func JsonDecoder(obj interface{}) cbhttp.MiddlewareFunc {
	unmarshaler := protojson.UnmarshalOptions{DiscardUnknown: true}
	return func(next cbhttp.RunnerFunc) cbhttp.RunnerFunc {
		return func(r *cbhttp.Request) (*cbhttp.Response, *lhttp.HttpError) {
			body, herr := next(r)
			if herr != nil {
				return nil, herr
			}

			if protoObj, ok := obj.(proto.Message); ok {
				var content bytes.Buffer
				if _, err := io.Copy(&content, body); err != nil {
					return nil, &lhttp.HttpError{Err: err}
				}
				if err := unmarshaler.Unmarshal(content.Bytes(), protoObj); err != nil {
					return nil, &lhttp.HttpError{Err: err}
				}
			} else {
				if err := json.NewDecoder(body).Decode(obj); err != nil {
					return nil, &lhttp.HttpError{Err: err}
				}
			}

			return nil, nil
		}
	}
}
