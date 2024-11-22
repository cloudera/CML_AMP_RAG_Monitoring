package interceptors_inflight

import (
	"context"
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"golang.org/x/sync/semaphore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
)

type Config struct {
	// Zero size means disabled and let everything through
	Size     uint64 `env:"INTERCEPTOR_LIMIT_INFLIGHT_SIZE" envDefault:"0"`
	Blocking bool   `env:"INTERCEPTOR_LIMIT_INFLIGHT_BLOCKING" envDefault:"true"`
}

func NewConfigFromEnv() (Config, error) {
	var cfg Config
	err := lconfig.Parse(&cfg)
	if err != nil {
		return cfg, err
	}

	return cfg, nil
}

type Interceptor struct {
	cfg Config
	sem *semaphore.Weighted
}

func NewInterceptor(cfg Config) *Interceptor {
	return &Interceptor{
		cfg: cfg,
		sem: semaphore.NewWeighted(int64(cfg.Size)),
	}
}

type checkResult struct {
	allowed bool
	err     error
	done    func()
}

func (interceptor *Interceptor) check(ctx context.Context) checkResult {
	result := checkResult{
		done: func() {},
	}
	if interceptor.cfg.Size > 0 {
		if !interceptor.cfg.Blocking {
			if !interceptor.sem.TryAcquire(1) {
				return result
			}
		} else {
			if err := interceptor.sem.Acquire(ctx, 1); err != nil {
				result.err = err
				return result
			}
		}
		result.done = func() {
			interceptor.sem.Release(1)
		}
	}
	result.allowed = true
	return result
}

func (interceptor *Interceptor) ToHTTP() sbhttpbase.MiddlewareFunc {
	return func(request *sbhttpbase.Request, next sbhttpbase.HandleFunc) {
		result := interceptor.check(request.Request.Context())
		defer result.done()
		if result.err != nil {
			request.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		if !result.allowed {
			request.Writer.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next(request)
	}
}

func (interceptor *Interceptor) ToGRPCStream() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		result := interceptor.check(ss.Context())
		defer result.done()
		if result.err != nil {
			return status.New(codes.Internal, "failed to acquire semaphore").Err()
		}
		if !result.allowed {
			return status.New(codes.ResourceExhausted, "too many requests").Err()
		}
		return handler(srv, ss)
	}
}
