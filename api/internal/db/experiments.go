package db

import (
	"context"
)

type Experiment struct {
	Id           int64
	ExperimentId string
}

type ExperimentService interface {
	ListExperiments(ctx context.Context) ([]*Experiment, error)
}
