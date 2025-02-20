package main

import (
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/config"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	postgresmig "github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/migrations/postgres"
	sqlitemig "github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/migrations/sqlite"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/server"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase"
	sbhttpserver "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/server"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
	lmigration "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql/migration"
)

type dependencies struct {
	cfg         *config.Config
	app         *app.Instance
	svc         *sbhttpserver.Instance
	swaggerApi  *server.SwaggerApiServer
	servers     []sbhttpserver.Server
	database    db.Database
	migration   *lmigration.Migration
	reconcilers *reconcilers.ReconcilerSet
	connections *clientbase.Connections
	dataStores  datasource.DataStores
}

func NewMigration(appCfg *config.Config, cfg *lsql.Config) (*lmigration.Migration, error) {
	if appCfg.Migrate {
		return lmigration.NewMigration(cfg, map[string]lmigration.MigrationSet{
			"sqlite":   {AssetNames: sqlitemig.AssetNames, Asset: sqlitemig.Asset},
			"postgres": {AssetNames: postgresmig.AssetNames, Asset: postgresmig.Asset},
		})
	}
	return nil, nil
}

func newDependencies(app *app.Instance, cfg *config.Config, svc *sbhttpserver.Instance,
	swaggerApi *server.SwaggerApiServer, servers []sbhttpserver.Server,
	database db.Database, migration *lmigration.Migration,
	connections *clientbase.Connections, dataStores datasource.DataStores,
	reconcilers *reconcilers.ReconcilerSet) *dependencies {
	return &dependencies{
		cfg:         cfg,
		app:         app,
		svc:         svc,
		swaggerApi:  swaggerApi,
		servers:     servers,
		database:    database,
		migration:   migration,
		reconcilers: reconcilers,
		connections: connections,
		dataStores:  dataStores,
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

	// Start the reconcilers
	deps.reconcilers.Start()
	defer deps.reconcilers.Finish()

	// Wait for the server to finish
	deps.app.WaitForFinish()
}
