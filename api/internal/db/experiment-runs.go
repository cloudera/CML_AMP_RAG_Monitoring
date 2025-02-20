package db

import (
	"context"
	"time"
)

type ExperimentRunService interface {
	CreateExperimentRun(ctx context.Context, run *ExperimentRun) (*ExperimentRun, error)
	GetExperimentRunById(ctx context.Context, id int64) (*ExperimentRun, error)
	GetExperimentRun(ctx context.Context, experimentId string, runId string) (*ExperimentRun, error)
	ListExperimentRuns(ctx context.Context, experimentId string) ([]*ExperimentRun, error)
	ListExperimentRunIdsForReconciliation(ctx context.Context, maxItems int64) ([]int64, error)
	ListExperimentRunIdsForMetricReconciliation(ctx context.Context, maxItems int64) ([]int64, error)
	MarkExperimentRunForMetricsReconciliation(ctx context.Context, id int64, reconcileMetrics bool) error
	MarkExperimentRunForReconciliation(ctx context.Context, id int64, reconcile bool) error
	DeleteExperimentRun(ctx context.Context, experimentId string, runId string) error
}

type ExperimentRun struct {
	Id           int64
	ExperimentId string
	RunId        string
	Created      bool
	Updated      bool
	Deleted      bool
	CreatedTs    time.Time
	UpdatedTs    time.Time
}
