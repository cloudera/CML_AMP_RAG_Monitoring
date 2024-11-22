package sbhttpserver

import (
	"context"
)

type NopServer struct{}

func (n *NopServer) Ready(ctx context.Context) error  { return nil }
func (n *NopServer) Live(ctx context.Context) error   { return nil }
func (n *NopServer) Shutdown() error                  { return nil }
func (n *NopServer) GetHandlers() []HandleDescription { return []HandleDescription{} }

var _ Server = &NopServer{}
