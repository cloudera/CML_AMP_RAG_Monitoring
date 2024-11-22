package interceptors_inflight

import (
	"context"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
	"testing"
)

func TestCore(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		cfg := Config{
			Size:     rapid.Uint64Range(0, 10).Draw(t, "size"),
			Blocking: rapid.Bool().Draw(t, "blocking"),
		}

		interceptor := NewInterceptor(cfg)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		nRequests := rapid.IntRange(0, 20).Draw(t, "n_requests")
		results := make([]checkResult, nRequests)
		for i := 0; i < nRequests; i++ {
			results[i] = interceptor.check(ctx)
		}

		defer func() {
			for i := 0; i < nRequests; i++ {
				results[i].done()
			}
		}()

		minAllowed := uint64(nRequests)
		if minAllowed > cfg.Size && cfg.Size != 0 {
			minAllowed = cfg.Size
		}

		countAllowed := uint64(0)
		countErrors := uint64(0)
		for i := 0; i < nRequests; i++ {
			if results[i].allowed {
				countAllowed++
			}
			if results[i].err != nil {
				countErrors++
			}
		}

		assert.Equal(t, minAllowed, countAllowed)
		if cfg.Blocking {
			assert.Equal(t, uint64(nRequests)-minAllowed, countErrors)
		} else {
			assert.Equal(t, uint64(0), countErrors)
		}
	})
}
