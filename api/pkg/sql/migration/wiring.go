package lmigration

import "github.com/google/wire"

var WireSet = wire.NewSet(NewMigration)
var TestingWireSet = wire.NewSet(NewMigrationSet, NewMigration)
