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
	UpdateRemoteRunId(ctx context.Context, id int64, remoteRunId string) error
	UpdateExperimentRunReconcileMetrics(ctx context.Context, id int64, reconcileMetrics bool) error
	UpdateExperimentRunUpdatedAndTimestamp(ctx context.Context, id int64, updated bool, updatedAt time.Time) error
	DeleteExperimentRun(ctx context.Context, experimentId string, runId string) error
}

type ExperimentRun struct {
	Id           int64
	ExperimentId string
	RunId        string
	RemoteRunId  string
	Created      bool
	Updated      bool
	Deleted      bool
	CreatedTs    time.Time
	UpdatedTs    time.Time
}
