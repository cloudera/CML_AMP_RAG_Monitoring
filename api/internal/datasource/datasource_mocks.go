package datasource

import (
	"context"
	"fmt"
)

type MlFlowMock struct {
	MetricsByRunId map[string][]Metric
}

var _ MetricStore = &MlFlowMock{}

func (m MlFlowMock) Metrics(_ context.Context, experimentId string, runId string) ([]Metric, error) {
	if runId == "" {
		return nil, fmt.Errorf("runId is required")
	}

	if metrics, ok := m.MetricsByRunId[runId]; ok {
		return metrics, nil
	}
	return nil, nil
}
