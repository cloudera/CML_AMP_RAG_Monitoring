package sbhttpserver

import (
	log "github.com/sirupsen/logrus"
	"net/http"
	"sync"
)

// Serve starts the service and wraps the given Server with instrumentation
func (b *Instance) Serve() error {
	// Register all the known handlers
	b.registerProfileHandlers()

	// Shut server down
	var wg sync.WaitGroup
	wg.Add(1)
	b.app.AddCloseFunc(func() error {
		b.server.Shutdown(b.app.Context())
		wg.Wait()
		return nil
	})

	log.Printf("serving at port %d", b.config.Port)
	go func() {
		defer wg.Done()
		err := b.server.ListenAndServe()

		if err != http.ErrServerClosed {
			log.Printf("failed to run server: %s", err)
			b.app.Stop(true)
		}
	}()

	return nil
}
