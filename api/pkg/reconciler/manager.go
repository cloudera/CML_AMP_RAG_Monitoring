package reconciler

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"sync"
	"time"
)

type Config struct {
	ResyncFrequency time.Duration
	MaxWorkers      int
	RunMaxItems     int
}

var ErrInvalidResyncFrequency = fmt.Errorf("invalid resync frequency")
var ErrInvalidMaxWorkers = fmt.Errorf("invalid max workers")
var ErrInvalidRunMaxItems = fmt.Errorf("invalid run max items")

func NewConfig(resyncFrequency time.Duration, maxWorkers, runMaxItems int) (*Config, error) {
	if resyncFrequency < 1*time.Millisecond {
		return nil, ErrInvalidResyncFrequency
	}
	if maxWorkers < 1 {
		return nil, ErrInvalidMaxWorkers
	}
	if runMaxItems < 1 {
		return nil, ErrInvalidRunMaxItems
	}
	return &Config{
		ResyncFrequency: resyncFrequency,
		MaxWorkers:      maxWorkers,
		RunMaxItems:     runMaxItems,
	}, nil
}

type Manager[T Key] struct {
	reconciler Reconciler[T]
	config     *Config
	context    context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	queue      *ReconcileQueue[T]
	tracer     trace.Tracer
}

func NewManager[T Key](ctx context.Context, cfg *Config, reconciler Reconciler[T]) *Manager[T] {
	if reconciler == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)

	tracer := otel.Tracer("reconciler_" + reconciler.Name())

	func() {
		ctx, span := startSpan(ctx, tracer, reconciler.Name()+".Reboot")
		defer span.End()

		reconciler.Reboot(ctx)
	}()

	return &Manager[T]{
		reconciler: reconciler,
		config:     cfg,
		context:    ctx,
		cancel:     cancel,
		queue:      NewReconcileQueue[T](),
		tracer:     tracer,
	}
}

func startSpan(ctx context.Context, tracer trace.Tracer, spanName string) (context.Context, trace.Span) {
	return tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
	)
}

func (r *Manager[T]) Start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()

		func() {
			ctx, span := startSpan(r.context, r.tracer, r.reconciler.Name()+".Resync")
			defer span.End()

			r.reconciler.Resync(ctx, r.queue)
		}()

		ticker := time.NewTicker(r.config.ResyncFrequency)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				func() {
					ctx, span := startSpan(r.context, r.tracer, r.reconciler.Name()+".Resync")
					defer span.End()

					r.reconciler.Resync(ctx, r.queue)
				}()
			case <-r.context.Done():
				log.Printf("reconciler Resync %s shutting down", r.reconciler.Name())
				return
			}
		}
	}()

	r.wg.Add(r.config.MaxWorkers)
	for i := 0; i < r.config.MaxWorkers; i++ {
		go func() {
			defer r.wg.Done()

			for {
				select {
				case <-r.context.Done():
					log.Printf("reconciler Reconcile %s shutting down", r.reconciler.Name())
					return
				default:
					items := r.queue.Pop(r.config.RunMaxItems)
					func() {
						ctx, span := startSpan(r.context, r.tracer, r.reconciler.Name()+".Reconcile")
						defer span.End()

						r.reconciler.Reconcile(ctx, items)
					}()
				}
			}
		}()
	}
}

func (r *Manager[T]) Finish() {
	r.queue.shutdown <- true
	r.cancel()
	r.wg.Wait()
}
