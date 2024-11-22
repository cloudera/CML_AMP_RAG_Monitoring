package main

import (
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/config"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	sqlitemig "github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/migrations/sqlite"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/server"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	sbhttpserver "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/server"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
	lmigration "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql/migration"
)

type dependencies struct {
	cfg               *config.Config
	app               *app.Instance
	svc               *sbhttpserver.Instance
	swaggerApi        *server.SwaggerApiServer
	servers           []sbhttpserver.Server
	database          db.Database
	migration         *lmigration.Migration
	metricsReconciler *reconciler.Manager[int64]
	connections       *clientbase.Connections
	dataStore         datasource.DataStore
}

func NewMigration(appCfg *config.Config, cfg *lsql.Config) (*lmigration.Migration, error) {
	if appCfg.Migrate {
		return lmigration.NewMigration(cfg, map[string]lmigration.MigrationSet{"sqlite": lmigration.MigrationSet{AssetNames: sqlitemig.AssetNames, Asset: sqlitemig.Asset}})
	}
	return nil, nil
}

func newDependencies(app *app.Instance, cfg *config.Config, svc *sbhttpserver.Instance,
	swaggerApi *server.SwaggerApiServer, servers []sbhttpserver.Server,
	database db.Database, migration *lmigration.Migration,
	connections *clientbase.Connections, dataStore datasource.DataStore,
	metricsReconciler *reconciler.Manager[int64]) *dependencies {
	return &dependencies{
		cfg:               cfg,
		app:               app,
		svc:               svc,
		swaggerApi:        swaggerApi,
		servers:           servers,
		database:          database,
		migration:         migration,
		metricsReconciler: metricsReconciler,
		connections:       connections,
		dataStore:         dataStore,
	}
}

func main() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	log.SetReportCaller(true)
	deps, err := InitializeDependencies()
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}

	if deps.cfg.Migrate {
		if err := deps.migration.Run(deps.cfg.MigrationVersion); err != nil {
			panic(err)
		}
	}

	servers := deps.servers

	if err := deps.svc.Register(sbhttpserver.NewMultiServer(servers)); err != nil {
		panic(err)
	}
	if err := deps.svc.Serve(); err != nil {
		panic(err)
	}

	// Start the metrics reconciler
	deps.metricsReconciler.Start()
	defer deps.metricsReconciler.Finish()

	// Wait for the server to finish
	deps.app.WaitForFinish()
}
