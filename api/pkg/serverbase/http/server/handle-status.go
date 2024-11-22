package sbhttpserver

import (
	log "github.com/sirupsen/logrus"
	"net/http"

	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

func (instance *Instance) registerStatusHandlers(server Server) {
	instance.RegisterHandler(&HandleDescription{
		Path:   "/_status/live",
		Method: "GET",
		Handler: sbhttpbase.HandleFunc(func(request *sbhttpbase.Request) {
			if err := server.Live(request.Request.Context()); err != nil {
				log.Printf("liveness request failed - %s", err)
				request.Writer.WriteHeader(http.StatusInternalServerError)
				request.Writer.Write([]byte(err.Error()))
			} else {
				request.Writer.WriteHeader(http.StatusOK)
			}
		}),
	})
	instance.RegisterHandler(&HandleDescription{
		Path:   "/_status/ready",
		Method: "GET",
		Handler: sbhttpbase.HandleFunc(func(request *sbhttpbase.Request) {
			if err := server.Ready(request.Request.Context()); err != nil {
				log.Printf("ready request failed - %s", err)
				request.Writer.WriteHeader(http.StatusInternalServerError)
				request.Writer.Write([]byte(err.Error()))
			} else {
				request.Writer.WriteHeader(http.StatusOK)
			}
		}),
	})
}
