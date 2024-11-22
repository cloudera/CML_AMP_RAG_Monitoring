package datasource

import "context"

type MetricStore interface {
	Metrics(ctx context.Context, runId string) ([]Metric, error)
}

type ArtifactStore interface {
	Artifacts(ctx context.Context, runId string, path *string) ([]Artifact, error)
	GetArtifact(ctx context.Context, runId string, path string) ([]byte, error)
}

type ExperimentStore interface {
	ListExperiments(ctx context.Context) ([]*Experiment, error)
	GetExperiment(ctx context.Context, experimentId string) (*Experiment, error)
}

type DataStore interface {
	MetricStore
	ArtifactStore
	ExperimentStore
}
