package builders

import (
	"github.com/google/wire"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase"
	cbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase/http"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http/portmap"
	interceptors_inflight "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/interceptors/in-flight"
	sbhttpserver "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/server"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
	lmigration "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql/migration"
	ltime "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/time"
)

var Builders = wire.NewSet(
	app.NewInstance,
	app.ContextFromInstance,
	clientbase.WireSet,
	cbhttp.NewConfigFromEnv,
	cbhttp.NewInstance,
	interceptors_inflight.NewConfigFromEnv,
	interceptors_inflight.NewInterceptor,
	lmigration.WireSet,
	lsql.WireSet,
	ltime.NewWallWatch,
	wire.Bind(new(ltime.Watch), new(ltime.WallWatch)),
	portmap.NewConfigFromEnv,
	sbhttpserver.NewConfigFromEnv,
	sbhttpserver.NewInstance,
)
