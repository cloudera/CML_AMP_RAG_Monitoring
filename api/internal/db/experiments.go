package db

import (
	"context"
	"time"
)

type Experiment struct {
	Id           int64
	ExperimentId string
	Name         string
	Created      bool
	Updated      bool
	Deleted      bool
	CreatedTs    time.Time
	UpdatedTs    time.Time
}

type ExperimentService interface {
	GetExperimentById(ctx context.Context, id int64) (*Experiment, error)
	GetExperimentByExperimentId(ctx context.Context, experimentId string) (*Experiment, error)
	ListExperiments(ctx context.Context) ([]*Experiment, error)
	ListExperimentIDsForReconciliation(ctx context.Context, maxItems int64) ([]int64, error)
	MarkExperimentIDForReconciliation(ctx context.Context, id int64) error
	UpdateExperimentCreatedAndTimestamp(ctx context.Context, id int64, created bool, ts time.Time) error
	UpdateExperimentUpdatedAndTimestamp(ctx context.Context, id int64, updated bool, ts time.Time) error
	CreateExperiment(ctx context.Context, experimentId string, name string, createdTs time.Time, updatedTs time.Time) (*Experiment, error)
}
