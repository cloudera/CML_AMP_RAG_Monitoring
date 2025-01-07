package db

import (
	"context"
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/models"
	"pgregory.net/rapid"
	"time"
)

type ExperimentRunsMock struct {
	ExperimentRuns []*ExperimentRun
}

func (e *ExperimentRunsMock) UpdateExperimentRunReconcileMetrics(ctx context.Context, id int64, reconcileMetrics bool) error {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentRunsMock) ListExperimentRunIdsForMetricReconciliation(ctx context.Context, maxItems int64) ([]int64, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentRunsMock) UpdateRemoteRunId(ctx context.Context, id int64, remoteRunId string) error {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentRunsMock) CreateExperimentRun(_ context.Context, run *ExperimentRun) (*ExperimentRun, error) {
	e.ExperimentRuns = append(e.ExperimentRuns, run)
	return run, nil
}

func (e *ExperimentRunsMock) GetExperimentRunById(ctx context.Context, id int64) (*ExperimentRun, error) {
	for _, run := range e.ExperimentRuns {
		if run.Id == id {
			return run, nil
		}
	}
	return nil, fmt.Errorf("run id %d not found", id)
}

func (e *ExperimentRunsMock) GetExperimentRun(ctx context.Context, experimentId string, runId string) (*ExperimentRun, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentRunsMock) ListExperimentRuns(_ context.Context, experimentId string) ([]*ExperimentRun, error) {
	var matches []*ExperimentRun
	for _, run := range e.ExperimentRuns {
		if run.ExperimentId == experimentId {
			matches = append(matches, run)
		}
	}
	return matches, nil
}

func (e *ExperimentRunsMock) ListExperimentRunIdsForReconciliation(_ context.Context, maxItems int64) ([]int64, error) {
	var runIds []int64
	for _, run := range e.ExperimentRuns {
		runIds = append(runIds, run.Id)
		if int64(len(runIds)) >= maxItems {
			break
		}
	}
	return runIds, nil
}

func (e *ExperimentRunsMock) UpdateExperimentRunUpdatedAndTimestamp(_ context.Context, id int64, updated bool, ts time.Time) error {
	for _, run := range e.ExperimentRuns {
		if run.Id == id {
			run.UpdatedTs = run.UpdatedTs.Add(1)
			return nil
		}
	}
	return fmt.Errorf("run id %d not found", id)
}

func (e *ExperimentRunsMock) DeleteExperimentRun(_ context.Context, experimentId string, runId string) error {
	// Delete the matching experiment run
	for i, run := range e.ExperimentRuns {
		if run.ExperimentId == experimentId && run.RunId == runId {
			e.ExperimentRuns = append(e.ExperimentRuns[:i], e.ExperimentRuns[i+1:]...)
			return nil
		}
	}
	return nil
}

var _ ExperimentRunService = &ExperimentRunsMock{}

type ExperimentsMock struct {
	Experiments []*Experiment
}

func (e *ExperimentsMock) GetExperimentById(ctx context.Context, id int64) (*Experiment, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentsMock) GetExperimentByExperimentId(ctx context.Context, experimentId string) (*Experiment, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentsMock) ListExperimentIDsForReconciliation(ctx context.Context, maxItems int64) ([]int64, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentsMock) MarkExperimentIDForReconciliation(ctx context.Context, id int64) error {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentsMock) UpdateRemoteExperimentId(ctx context.Context, id int64, remoteExperimentId string) error {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentsMock) UpdateExperimentCreatedAndTimestamp(ctx context.Context, id int64, created bool, ts time.Time) error {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentsMock) UpdateExperimentUpdatedAndTimestamp(ctx context.Context, id int64, updated bool, ts time.Time) error {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentsMock) CreateExperiment(ctx context.Context, experimentId string, createdTs time.Time, updatedTs time.Time) (*Experiment, error) {
	//TODO implement me
	panic("implement me")
}

func (e *ExperimentsMock) ListExperiments(_ context.Context) ([]*Experiment, error) {
	return e.Experiments, nil
}

var _ ExperimentService = &ExperimentsMock{}

type MetricsMock struct {
	CreatedMetrics []*Metric
}

func (mm *MetricsMock) CreateMetric(_ context.Context, m *Metric) (*Metric, error) {
	now := time.Now()
	m.Timestamp = &now
	mm.CreatedMetrics = append(mm.CreatedMetrics, m)
	return m, nil
}

func (mm *MetricsMock) GetMetric(ctx context.Context, id int64) (*Metric, error) {
	for _, metric := range mm.CreatedMetrics {
		if metric.Id == id {
			return metric, nil
		}
	}
	return nil, fmt.Errorf("metric id %d not found", id)
}

func (mm *MetricsMock) ListMetrics(_ context.Context, experimentId *string, runIds []string, metricNames []string) ([]*Metric, error) {
	var matches []*Metric
	for _, metric := range mm.CreatedMetrics {
		if experimentId != nil && metric.ExperimentId != *experimentId {
			continue
		}
		if len(runIds) > 0 {
			found := false
			for _, runId := range runIds {
				if metric.RunId == runId {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		if len(metricNames) > 0 {
			found := false
			for _, metricName := range metricNames {
				if metric.Name == metricName {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		matches = append(matches, metric)
	}
	return matches, nil
}

var _ MetricsService = &MetricsMock{}

var nextExperimentId int64 = 0
var nextExperimentRunId int64 = 0

func ExperimentGenerator() *rapid.Generator[*Experiment] {
	id := nextExperimentId
	nextExperimentId++
	return rapid.Custom(func(t *rapid.T) *Experiment {
		return &Experiment{
			Id:           id,
			ExperimentId: rapid.StringMatching("[a-z0-9A-Z]{16}").Draw(t, "experimentId"),
		}
	})
}
func ExperimentRunGenerator() *rapid.Generator[*ExperimentRun] {
	return rapid.Custom(func(t *rapid.T) *ExperimentRun {
		id := nextExperimentRunId
		nextExperimentRunId++
		return &ExperimentRun{
			Id:           id,
			ExperimentId: rapid.StringMatching("[a-z0-9A-Z]{16}").Draw(t, "experimentId"),
			RunId:        rapid.StringMatching("[a-z0-9A-Z]{16}").Draw(t, "runId"),
			Created:      rapid.Bool().Draw(t, "created"),
			Updated:      rapid.Bool().Draw(t, "updated"),
			Deleted:      rapid.Bool().Draw(t, "deleted"),
		}
	})
}

func MetricTagGenerator() *rapid.Generator[*models.MetricTag] {
	return rapid.Custom(func(t *rapid.T) *models.MetricTag {
		return &models.MetricTag{
			Key:   rapid.String().Draw(t, "key"),
			Value: rapid.String().Draw(t, "value"),
		}
	})
}

func MetricValueGenerator() *rapid.Generator[*models.MetricValue] {
	return rapid.Custom(func(t *rapid.T) *models.MetricValue {
		return &models.MetricValue{
			MetricType:   rapid.String().Draw(t, "type"),
			NumericValue: rapid.Float64().Draw(t, "numericValue"),
			StringValue:  rapid.String().Draw(t, "stringValue"),
		}
	})
}
func MetricGenerator() *rapid.Generator[*models.Metric] {
	return rapid.Custom(func(t *rapid.T) *models.Metric {
		return &models.Metric{
			ExperimentID:    rapid.StringMatching("[a-z0-9A-Z]{16}").Draw(t, "experimentId"),
			ExperimentRunID: rapid.StringMatching("[a-z0-9A-Z]{16}").Draw(t, "runId"),
			Name:            rapid.String().Draw(t, "name"),
			Tags:            rapid.SliceOf(MetricTagGenerator()).Draw(t, "tags"),
			Ts:              strfmt.NewDateTime(), // TODO
			Value:           MetricValueGenerator().Draw(t, "value"),
		}
	})
}
