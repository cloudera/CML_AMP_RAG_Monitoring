package metrics

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	"pgregory.net/rapid"
	"testing"
)

type state struct {
	experiments    *db.ExperimentsMock
	runs           *db.ExperimentRunsMock
	metrics        *db.MetricsMock
	mlFlow         datasource.MlFlowMock
	reconciler     *Reconciler
	config         *Config
	queue          *reconciler.ReconcileQueue[int64]
	reconcileItems []reconciler.ReconcileItem[int64]
}

func newTestState(t *rapid.T) (*state, error) {
	config, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
		return nil, err
	}

	experiments := db.ExperimentsMock{}

	experimentRuns := rapid.SliceOf(db.ExperimentRunGenerator()).Draw(t, "experimentRuns")

	runs := db.ExperimentRunsMock{
		ExperimentRuns: experimentRuns,
	}

	mlFlow := datasource.MlFlowMock{
		MetricsByRunId: make(map[string][]datasource.Metric),
	}

	for range rapid.UintRange(0, uint(len(experimentRuns))).Draw(t, "runs") {
		run := rapid.SampledFrom(experimentRuns).Draw(t, "runsWithMetrics")
		metric := datasource.Metric{
			Key:       rapid.StringMatching(".{1,5}").Draw(t, "key"),
			Value:     rapid.Float64().Draw(t, "value"),
			Timestamp: rapid.Int64().Draw(t, "timestamp"),
			Step:      rapid.Int().Draw(t, "step"),
		}
		mlFlow.MetricsByRunId[run.RunId] = append(mlFlow.MetricsByRunId[run.RunId], metric)
	}

	metrics := &db.MetricsMock{}

	database := db.NewDatabase(&experiments, &runs, metrics)
	r := NewReconciler(config, database, &mlFlow)
	return &state{
		experiments: &experiments,
		runs:        &runs,
		metrics:     metrics,
		mlFlow:      mlFlow,
		reconciler:  r,
		config:      config,
		queue:       reconciler.NewReconcileQueue[int64](),
	}, nil
}

func TestReconcilerResync(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		state, err := newTestState(rt)
		if err != nil {
			rt.Fatalf("failed to create test state: %v", err)
			return
		}

		state.reconciler.Resync(context.TODO(), state.queue)

		target := len(state.runs.ExperimentRuns)
		if target > state.config.ResyncMaxItems {
			target = state.config.ResyncMaxItems
		}
		// Property: The number of items in the queue should be the minimum of the number of runIds and the resync max items
		assert.Equal(t, len(state.queue.Pending), target)

		// TODO property for retry
	})
}

func TestReconcilerReconcile(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		state, err := newTestState(rt)
		if err != nil {
			rt.Fatalf("failed to create test state: %v", err)
			return
		}

		r := state.reconciler
		// Moves ids into Pending
		r.Resync(context.TODO(), state.queue)

		maxPop := len(state.queue.Pending)
		if maxPop > 0 {
			// We need more than 0 items or it will block
			// TODO async testing
			state.reconcileItems = state.queue.Pop(maxPop)
			if len(state.reconcileItems) > 0 {
				r.Reconcile(context.TODO(), state.reconcileItems)

				// Property: the number of runs metrics by run id equals the number of created metrics
				for runId, metrics := range state.mlFlow.MetricsByRunId {
					numCreated := 0
					for _, metric := range state.metrics.CreatedMetrics {
						if metric.RunId == runId {
							numCreated++
						}
					}
					if !assert.Equal(t, len(metrics), numCreated) {
						println("fail")
					}
				}
			}
		}
	})
}
