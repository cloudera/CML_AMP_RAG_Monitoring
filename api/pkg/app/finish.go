package app

import (
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type CloseFunc func() error

func (instance *Instance) AddCloseFunc(fn CloseFunc) {
	instance.AddCloser(&closeWrapper{fn: fn})
}

type closeWrapper struct {
	fn CloseFunc
}

func (w *closeWrapper) Close() error {
	return w.fn()
}

func (instance *Instance) AddCloser(closer io.Closer) {
	instance.closers = append(instance.closers, closer)
}

func (instance *Instance) Stop(failed bool) {
	instance.failed = failed || instance.failed
	close(instance.stop)
}

func (instance *Instance) WaitForFinish() {
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)
		select {
		case <-sigint:
		case <-instance.stop:
		}

		instance.cancel()

		var wg sync.WaitGroup
		wg.Add(len(instance.closers))
		for i := range instance.closers {
			go func(i int) {
				defer wg.Done()
				if err := instance.closers[i].Close(); err != nil {
					//instance.logger.Error("failed to close", zap.Error(err))
					instance.failed = true
				}
			}(i)
		}
		wg.Wait()

		if instance.failed {
			os.Exit(1)
		}

		close(instance.stop)
	}()

	// Wait until everything is done and finished
	select {
	case <-instance.stop:
	}
}
