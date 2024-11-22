package sbhttpserver

import (
	"context"
	"fmt"
)

type MultiServer struct {
	servers []Server
}

func NewMultiServer(servers []Server) *MultiServer {
	return &MultiServer{
		servers: servers,
	}
}

func (s *MultiServer) Ready(ctx context.Context) error {
	for _, ss := range s.servers {
		if err := ss.Ready(ctx); err != nil {
			return err
		}
	}
	return nil
}
func (s *MultiServer) Live(ctx context.Context) error {
	for _, ss := range s.servers {
		if err := ss.Live(ctx); err != nil {
			return err
		}
	}
	return nil
}
func (s *MultiServer) Shutdown() error {
	errors := make([]error, 0)
	for _, ss := range s.servers {
		if err := ss.Shutdown(); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		errorMessage := ""
		for _, err := range errors {
			errorMessage += err.Error() + "\n"
		}
		return fmt.Errorf(errorMessage)
	}
	return nil
}

func (s *MultiServer) GetHandlers() []HandleDescription {
	handlers := make([]HandleDescription, 0)
	for _, ss := range s.servers {
		handlers = append(handlers, ss.GetHandlers()...)
	}
	return handlers
}
