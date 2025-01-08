package datasource

import (
	"context"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase"
	"time"
)

type MetricStore interface {
	Metrics(ctx context.Context, experimentId string, runId string) ([]Metric, error)
}

type ArtifactStore interface {
	Artifacts(ctx context.Context, runId string, path *string) ([]Artifact, error)
	GetArtifact(ctx context.Context, runId string, path string) ([]byte, error)
}

type ExperimentStore interface {
	ListExperiments(ctx context.Context, maxItems int64, pageToken string) ([]*Experiment, error)
	GetExperiment(ctx context.Context, experimentId string) (*Experiment, error)
	CreateExperiment(ctx context.Context, name string) (string, error)
}

type RunStore interface {
	GetRun(ctx context.Context, experimentId string, runId string) (*Run, error)
	ListRuns(ctx context.Context, experimentId string) ([]*Run, error)
	CreateRun(ctx context.Context, experimentId string, name string, createdTs time.Time, tags []RunTag) (string, error)
	UpdateRun(ctx context.Context, run *Run) (*Run, error)
}

type DataStore interface {
	MetricStore
	ArtifactStore
	ExperimentStore
	RunStore
}

type DataStores struct {
	Local  DataStore
	Remote DataStore
}

func NewDataStores(cfg *Config, connections *clientbase.Connections) DataStores {
	return DataStores{
		Local:  NewMLFlow(cfg.LocalMLFlowBaseUrl, cfg, connections),
		Remote: NewPlatformMLFlow(cfg.CDSWMLFlowBaseUrl, cfg, connections),
	}
}
