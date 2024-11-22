package app

import (
	"context"
	"io"
)

type Instance struct {
	closers []io.Closer
	failed  bool
	stop    chan bool
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewInstance() *Instance {
	ctx, cancel := context.WithCancel(context.Background())
	return &Instance{
		stop:   make(chan bool),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (instance *Instance) Context() context.Context {
	return instance.ctx
}

func ContextFromInstance(instance *Instance) context.Context {
	return instance.ctx
}
