package datasource

type Metric struct {
	Key       string  `json:"key"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
	Step      int     `json:"step"`
}

type Artifact struct {
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	FileSize int64  `json:"file_size"`
}

type ExperimentTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Experiment struct {
	Id               int64           `json:"id"`
	Name             string          `json:"name"`
	ArtifactLocation string          `json:"artifact_location"`
	LifecycleStage   string          `json:"lifecycle_stage"`
	LastUpdatedTime  int64           `json:"last_updated_time"`
	CreatedTime      int64           `json:"created_time"`
	Tags             []ExperimentTag `json:"tags"`
}
