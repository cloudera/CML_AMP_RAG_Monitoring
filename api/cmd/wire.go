//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/config"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db/postgres"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers"
	recexperiments "github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/experiments"
	recmetrics "github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/metrics"
	recruns "github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/runs"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/restapi"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/server"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase/http"
	sbhttpserver "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/server"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
)

// wire up the dependencies.
func InitializeDependencies() (*dependencies, error) {
	wire.Build(config.NewConfigFromEnv, app.NewInstance,
		cbhttp.NewConfigFromEnv, cbhttp.NewInstance, clientbase.NewConfigFromEnv, clientbase.NewConnections,
		sbhttpserver.NewConfigFromEnv, sbhttpserver.NewInstance,
		server.NewSwaggerConfig, server.NewHandler, server.NewHttpServers,
		lsql.NewConfigFromEnv, postgres.NewInstance, postgres.NewExperiments, postgres.NewExperimentRuns,
		postgres.NewMetrics, postgres.NewDatabase, NewMigration,
		restapi.NewMetricsAPI, restapi.NewExperimentRunsAPI, restapi.NewExperimentAPI,
		server.NewSwaggerApiServer,
		datasource.NewConfigFromEnv, datasource.NewDataStores,
		recexperiments.NewConfigFromEnv, recexperiments.NewExperimentReconciler, recexperiments.NewExperimentRunReconciler,
		recruns.NewConfigFromEnv, recruns.NewRunReconciler,
		recmetrics.NewConfigFromEnv, recmetrics.NewMetricsReconciler,
		reconcilers.NewReconcilerSet,
		newDependencies)
	return &dependencies{}, nil
}
