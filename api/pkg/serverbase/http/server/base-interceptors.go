package sbhttpserver

import (
	lconfig "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/config"
	lgzip "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/gzip"
	interceptors_inflight "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/interceptors/in-flight"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

type BaseInterceptorsConfig struct {
	DisableGzipRequestDecompression bool
	DisableGzipResponseCompression  bool
	DisableRequestLog               bool
	DisableAuthExtraction           bool
	DisableMetrics                  bool
	LatencyBuckets                  []float64 `env:"SERVER_HTTP_LATENCY_BUCKETS" envDefault:"0.01 0.03 0.1 0.3 1 3 10" envSeparator:" "`
	DisableTracing                  bool
	DisableLimiter                  bool
	DisableDefaultJsonContentType   bool
}

var baseDefault BaseInterceptorsConfig

func init() {
	lconfig.MustParse(&baseDefault)
}

func GetBaseInterceptors(cfg BaseInterceptorsConfig, limiter *interceptors_inflight.Interceptor) []sbhttpbase.RegistrableMiddleware {
	if cfg.LatencyBuckets == nil {
		cfg.LatencyBuckets = baseDefault.LatencyBuckets
	}

	ret := []sbhttpbase.RegistrableMiddleware{}

	if !cfg.DisableLimiter && limiter != nil {
		ret = append(ret, limiter.ToHTTP())
	}

	if !cfg.DisableGzipRequestDecompression {
		ret = append(ret, lgzip.HttpServerDecompressRequestInterceptor())
	}

	if !cfg.DisableGzipResponseCompression {
		ret = append(ret, lgzip.HttpServerCompressResponseInterceptor())
	}
	return ret
}
