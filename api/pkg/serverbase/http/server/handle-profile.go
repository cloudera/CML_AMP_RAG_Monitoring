package sbhttpserver

import (
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
	"net/http/pprof"
)

func (b *Instance) registerProfileHandlers() {
	log.Printf("registering profile handlers")

	b.RegisterHandler(&HandleDescription{
		Path:    "/debug/pprof/",
		Method:  "GET",
		Handler: sbhttpbase.HandleStdFunc(pprof.Index),
	})
	for _, subcomponent := range []string{"allocs", "block", "goroutine", "heap", "mutex", "threadcreate"} {
		b.RegisterHandler(&HandleDescription{
			Path:    "/debug/pprof/" + subcomponent,
			Method:  "GET",
			Handler: sbhttpbase.HandleStdFunc(pprof.Index),
		})
	}
	b.RegisterHandler(&HandleDescription{
		Path:    "/debug/pprof/cmdline",
		Method:  "GET",
		Handler: sbhttpbase.HandleStdFunc(pprof.Cmdline),
	})
	b.RegisterHandler(&HandleDescription{
		Path:    "/debug/pprof/profile",
		Method:  "GET",
		Handler: sbhttpbase.HandleStdFunc(pprof.Profile),
	})
	b.RegisterHandler(&HandleDescription{
		Path:    "/debug/pprof/symbol",
		Method:  "GET",
		Handler: sbhttpbase.HandleStdFunc(pprof.Symbol),
	})
	b.RegisterHandler(&HandleDescription{
		Path:    "/debug/pprof/trace",
		Method:  "GET",
		Handler: sbhttpbase.HandleStdFunc(pprof.Trace),
	})
}
