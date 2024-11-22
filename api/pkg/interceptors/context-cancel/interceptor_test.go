package context_cancel

import (
	"context"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/wrappers"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"net/http"
	"testing"
	"time"
)

func TestCaptureContextCancel(t *testing.T) {
	tables := []struct {
		mainSleep   time.Duration
		callerSleep time.Duration
		cancel      bool
		code        int
	}{
		{time.Millisecond, 10 * time.Millisecond, true, 200},
		{time.Millisecond, 10 * time.Millisecond, false, 200},
		{10 * time.Millisecond, time.Millisecond, true, 499},
		{10 * time.Millisecond, time.Millisecond, false, 499},
	}

	for _, table := range tables {
		next := func(request *sbhttpbase.Request) {
			time.Sleep(table.mainSleep)
			request.Writer.WriteHeader(200)
		}

		capturer := wrappers.CustomizableResponseWriter{}

		var ctx context.Context

		if table.cancel {
			ctx2, cancel := context.WithCancel(context.Background())
			ctx = ctx2

			go func() {
				time.Sleep(table.callerSleep)
				cancel()
			}()
		} else {
			ctx2, cancel := context.WithTimeout(context.Background(), table.callerSleep)
			ctx = ctx2
			defer cancel()
		}

		request := &sbhttpbase.Request{
			Writer:  &capturer,
			Request: (&http.Request{}).WithContext(ctx),
		}

		Interceptor{}.ToHTTP()(request, next)

		if capturer.Code != table.code {
			t.Errorf("Got code %d and expected %d", capturer.Code, table.code)
		}
	}
}
