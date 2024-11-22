package swaggerinterceptors

import (
	"context"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"net/http"
)

type UnaryServerInfo struct {
	FullMethod string
}

type UnaryHandler func(ctx context.Context, header http.Header, req interface{}) (interface{}, *lhttp.HttpError)

type UnaryServerInterceptor func(ctx context.Context, header http.Header, req interface{}, info *UnaryServerInfo, handler UnaryHandler) (resp interface{}, err *lhttp.HttpError)

//type UnaryInvoker func(ctx context.Context, method string, req interface{}) (interface{}, *lhttp.HttpError)
//
//type UnaryClientInterceptor func(ctx context.Context, method string, req interface{}, invoker UnaryInvoker) (reply interface{}, err *lhttp.HttpError)
