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
	ExperimentId     string          `json:"experiment_id"`
	Name             string          `json:"name"`
	ArtifactLocation string          `json:"artifact_location"`
	LifecycleStage   string          `json:"lifecycle_stage"`
	LastUpdatedTime  int64           `json:"last_update_time"`
	CreatedTime      int64           `json:"creation_time"`
	Tags             []ExperimentTag `json:"tags"`
}

type RunInfo struct {
	RunId          string    `json:"run_id"`
	Name           string    `json:"run_name"`
	ExperimentId   string    `json:"experiment_id"`
	Status         RunStatus `json:"status"`
	StartTime      int64     `json:"start_time"`
	EndTime        int64     `json:"end_time"`
	ArtifactUri    string    `json:"artifact_uri"`
	LifecycleStage string    `json:"lifecycle_stage"`
}

type RunStatus string

const (
	RunStatusRunning   RunStatus = "RUNNING"
	RunStatusScheduled RunStatus = "SCHEDULED"
	RunStatusFinished  RunStatus = "FINISHED"
	RunStatusFailed    RunStatus = "FAILED"
	RunStatusKilled    RunStatus = "KILLED"
)

type Param struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RunTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type RunData struct {
	Metrics []Metric   `json:"metrics"`
	Params  []Param    `json:"params"`
	Tags    []RunTag   `json:"tags"`
	Files   []Artifact `json:"files"`
}

type Run struct {
	Info RunInfo `json:"info"`
	Data RunData `json:"data"`
}

type RunResponse struct {
	Run Run `json:"run"`
}

type RunsResponse struct {
	Runs          []Run  `json:"runs"`
	NextPageToken string `json:"next_page_token"`
}

type ArtifactsResponse struct {
	RootUri       string     `json:"root_uri"`
	Files         []Artifact `json:"files"`
	NextPageToken string     `json:"next_page_token"`
}

type ExperimentResponse struct {
	Experiment Experiment `json:"experiment"`
}

type ExperimentListResponse struct {
	Experiments   []*Experiment `json:"experiments"`
	NextPageToken string        `json:"next_page_token"`
}
