package clientbase

import "github.com/google/wire"

var WireSet = wire.NewSet(
	NewConfigFromEnv,
	NewConnections,
)
