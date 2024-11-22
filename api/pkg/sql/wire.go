//go:build wireinject
// +build wireinject

package lsql

import (
	"github.com/google/wire"
	ltest "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/test"
)

func initializeTest(t ltest.T) (*Instance, error) {
	wire.Build(
		TestingWireSet,
	)
	return &Instance{}, nil
}
