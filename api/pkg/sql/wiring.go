package lsql

import "github.com/google/wire"

var WireSet = wire.NewSet(NewConfigFromEnv, NewInstance)
var TestingWireSet = wire.NewSet(NewTestingConfig, NewInstance)
