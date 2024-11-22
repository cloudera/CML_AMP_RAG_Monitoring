package context_cancel

import (
	"context"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Interceptor struct{}

func (interceptor Interceptor) ToHTTP() sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		wrapper := wrappers.CustomizableResponseWriter{
			Response: request.Writer,
			OnWriteHeader: func(w *wrappers.CustomizableResponseWriter, code int) {
				if err := request.Request.Context().Err(); err != nil {
					if err == context.Canceled || err == context.DeadlineExceeded {
						code = 499
					}
				}
				request.Writer.WriteHeader(code)
			},
		}

		next(request.WithWriter(&wrapper))
	}
}

func (interceptor Interceptor) ToGRPCStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, &stream{ss})
	}
}

type stream struct {
	grpc.ServerStream
}

func (s *stream) SetHeader(h metadata.MD) error {
	err := s.ServerStream.SetHeader(h)
	if err == context.Canceled || err == context.DeadlineExceeded {
		return nil
	}
	return err
}
func (s *stream) SendHeader(h metadata.MD) error {
	err := s.ServerStream.SendHeader(h)
	if err == context.Canceled || err == context.DeadlineExceeded {
		return nil
	}
	return err
}
func (s *stream) SendMsg(m interface{}) error {
	err := s.ServerStream.SendMsg(m)
	if err == context.Canceled || err == context.DeadlineExceeded {
		return nil
	}
	return err
}
func (s *stream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == context.Canceled || err == context.DeadlineExceeded {
		return nil
	}
	return err
}
