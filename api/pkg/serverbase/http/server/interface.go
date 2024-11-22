package sbhttpserver

import (
	"context"
	sbhttpbase "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/serverbase/http/base"
)

// HandleDescription is the set of requirements to describe a handle
type HandleDescription struct {
	NotFound   bool
	Path       string
	Method     string
	Handler    sbhttpbase.HandleFunc
	Middleware []sbhttpbase.RegistrableMiddleware
}

type ReadinessProvider interface {
	// Method to verify the ready status of the container
	Ready(ctx context.Context) error
}

type LivenessProvider interface {
	// Method to verify the live status of the container
	Live(ctx context.Context) error
}

type ShutdownProvider interface {
	// Method to shutdown the container
	Shutdown() error
}

// Server is an interface that every implementation of the server has to provide
type Server interface {
	ReadinessProvider
	LivenessProvider
	ShutdownProvider
	// Mapping of handling paths
	GetHandlers() []HandleDescription
}
