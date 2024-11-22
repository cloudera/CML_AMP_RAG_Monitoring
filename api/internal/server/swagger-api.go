package server

import (
	"context"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/config"
	restimpl "github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/restapi"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	sbhttpserver "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/server"
	sbswagger "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/swagger"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi"
	"net/http"
)

type SwaggerApiServer struct {
	app *app.Instance
	cfg *config.Config
	db  *lsql.Instance
}

func NewSwaggerConfig(metricsApi *restimpl.MetricsAPI, runsApi *restimpl.ExperimentRunsAPI, experimentsApi *restimpl.ExperimentAPI) (restapi.Config, error) {
	return restapi.Config{
		MetricsAPI:     metricsApi,
		RunsAPI:        runsApi,
		ExperimentsAPI: experimentsApi,
	}, nil
}

func NewHandler(cfg restapi.Config) (http.Handler, error) {
	return restapi.Handler(cfg)
}

func NewSwaggerApiServer(app *app.Instance, cfg *config.Config, db *lsql.Instance) *SwaggerApiServer {
	return &SwaggerApiServer{
		app: app,
		cfg: cfg,
		db:  db,
	}
}

func NewHttpServers(cfg restapi.Config, handler http.Handler, apiServer *SwaggerApiServer) []sbhttpserver.Server {
	spec, err := sbswagger.Merge(restapi.SwaggerJSON)
	if err != nil {
		panic(err)
	}
	return []sbhttpserver.Server{
		sbswagger.New(cfg, handler, spec),
		apiServer,
	}
}

// Ready fails if we cannot ping the database in a reasonable time
func (s *SwaggerApiServer) Ready(ctx context.Context) error {
	if err := s.db.Ping(ctx); err != nil {
		return err
	}
	return nil
}

// Live doesn't do any check. Just answering the request is enough evidence we're alive
func (s *SwaggerApiServer) Live(ctx context.Context) error {
	return nil
}

func (s *SwaggerApiServer) Shutdown() error {
	return nil
}

func (s *SwaggerApiServer) GetHandlers() []sbhttpserver.HandleDescription {
	return []sbhttpserver.HandleDescription{}
}
