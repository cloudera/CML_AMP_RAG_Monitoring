package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/util"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase"
	cbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase/http"
	"io"
	"strconv"
	"time"
)

type PlatformExperiment struct {
	Id               string          `json:"id"`
	ProjectId        string          `json:"project_id"`
	Name             string          `json:"name"`
	ArtifactLocation string          `json:"artifact_location"`
	LifecycleStage   string          `json:"lifecycle_stage"`
	LastUpdatedTime  int64           `json:"last_update_time"`
	CreatedTime      int64           `json:"creation_time"`
	Tags             []ExperimentTag `json:"tags"`
}

type PlatformExperimentListResponse struct {
	Experiments   []PlatformExperiment `json:"experiments"`
	NextPageToken string               `json:"next_page_token"`
}

type PlatformRun struct {
	Id             string          `json:"id"`
	Name           string          `json:"run_name"`
	Status         string          `json:"status"`
	StartTime      time.Time       `json:"start_time"`
	EndTime        time.Time       `json:"end_time"`
	ArtifactUri    string          `json:"artifact_uri"`
	LifecycleStage string          `json:"lifecycle_stage"`
	Data           PlatformRunData `json:"data"`
}

type PlatformMetric struct {
	Key       string    `json:"key"`
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Step      string    `json:"step"`
}

type PlatformRunData struct {
	Metrics []PlatformMetric `json:"metrics"`
	Params  []Param          `json:"params"`
	Tags    []RunTag         `json:"tags"`
}

type PlatformMLFlow struct {
	MLFlow
}

var _ DataStore = &PlatformMLFlow{}

func NewPlatformMLFlow(baseUrl string, cfg *Config, connections *clientbase.Connections) DataStore {
	return &PlatformMLFlow{
		MLFlow: MLFlow{
			baseUrl:     baseUrl,
			cfg:         cfg,
			connections: connections,
		},
	}
}

func (m *PlatformMLFlow) UpdateRun(ctx context.Context, run *Run) (*Run, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s/runs/%s", m.baseUrl, m.cfg.CDSWProjectID, run.Info.ExperimentId, run.Info.RunId)
	req := cbhttp.NewRequest(ctx, "PATCH", url)

	params := make([]Param, 0)
	for _, param := range run.Data.Params {
		var val = param.Value
		if len(val) > 250 {
			log.Debugf("param %s value is too long for platform MLFlow, truncating", param.Key)
			val = val[:250]
		}
		params = append(params, Param{
			Key:   param.Key,
			Value: val,
		})
	}

	data := PlatformRunData{
		Metrics: make([]PlatformMetric, 0),
		Params:  params,
		Tags:    run.Data.Tags,
	}

	for _, metric := range run.Data.Metrics {
		data.Metrics = append(data.Metrics, PlatformMetric{
			Key:       metric.Key,
			Value:     metric.Value,
			Timestamp: util.TimeStamp(metric.Timestamp),
			Step:      strconv.Itoa(metric.Step),
		})
	}

	var status string
	switch run.Info.Status {
	case RunStatusRunning:
		status = "EXPERIMENT_RUN_RUNNING"
	case RunStatusScheduled:
		status = "EXPERIMENT_RUN_SCHEDULED"
	case RunStatusFinished:
		status = "EXPERIMENT_RUN_FINISHED"
	case RunStatusFailed:
		status = "EXPERIMENT_RUN_FAILED"
	case RunStatusKilled:
		status = "EXPERIMENT_RUN_KILLED"
	}

	platformRun := PlatformRun{
		Id:             run.Info.RunId,
		Name:           run.Info.Name,
		Status:         status,
		EndTime:        time.UnixMilli(run.Info.EndTime),
		LifecycleStage: run.Info.LifecycleStage,
		Data:           data,
	}

	encoded, serr := json.Marshal(platformRun)
	if serr != nil {
		log.Printf("failed to encode body: %s", serr)
		return nil, serr
	}
	req.Body = io.NopCloser(bytes.NewReader(encoded))
	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", m.cfg.CDSWApiKey))
	resp, lerr := m.connections.HttpClient.Do(req)
	if lerr != nil {
		log.Printf("failed to update run %s: %s", run.Info.RunId, lerr)
		return nil, lerr
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to update run %s: %s", run.Info.RunId, resp.Status)
	}
	buff, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		return nil, ioerr
	}
	var updatedRun PlatformRun
	jerr := json.Unmarshal(buff, &updatedRun)
	if jerr != nil {
		return nil, jerr
	}
	return &Run{
		Info: RunInfo{
			RunId:          updatedRun.Id,
			Name:           updatedRun.Name,
			ExperimentId:   run.Info.ExperimentId,
			Status:         RunStatus(updatedRun.Status),
			StartTime:      updatedRun.StartTime.UnixMilli(),
			EndTime:        updatedRun.EndTime.UnixMilli(),
			ArtifactUri:    updatedRun.ArtifactUri,
			LifecycleStage: updatedRun.LifecycleStage,
		},
		Data: RunData{},
	}, nil
}

func (m *PlatformMLFlow) GetRun(ctx context.Context, experimentId string, runId string) (*Run, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s/runs/%s", m.baseUrl, m.cfg.CDSWProjectID, experimentId, runId)
	req := cbhttp.NewRequest(ctx, "GET", url)

	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", m.cfg.CDSWApiKey))
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Printf("failed to fetch run %s: %s", runId, err)
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, nil
	}
	defer resp.Body.Close()

	body, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		return nil, err
	}

	var run PlatformRun
	jerr := json.Unmarshal(body, &run)
	if jerr != nil {
		return nil, jerr
	}
	data := RunData{
		Metrics: make([]Metric, 0),
		Params:  run.Data.Params,
		Tags:    run.Data.Tags,
	}
	for _, metric := range run.Data.Metrics {
		step, err := strconv.Atoi(metric.Step)
		if err != nil {
			log.Printf("failed to convert step to int: %s", err)
			return nil, err
		}
		data.Metrics = append(data.Metrics, Metric{
			Key:       metric.Key,
			Value:     metric.Value,
			Timestamp: metric.Timestamp.Unix(),
			Step:      step,
		})
	}
	return &Run{
		Info: RunInfo{
			RunId:          runId,
			Name:           run.Name,
			ExperimentId:   experimentId,
			Status:         RunStatus(run.Status),
			StartTime:      run.StartTime.UnixMilli(),
			EndTime:        run.EndTime.UnixMilli(),
			ArtifactUri:    run.ArtifactUri,
			LifecycleStage: run.LifecycleStage,
		},
		Data: data,
	}, nil
}

func (m *PlatformMLFlow) ListRuns(ctx context.Context, experimentId string) ([]*Run, error) {
	token := ""
	done := false
	runs := make([]*Run, 0)
	for {
		if done {
			break
		}
		url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s/runs?page_token=%s", m.baseUrl, m.cfg.CDSWProjectID, experimentId, token)
		req := cbhttp.NewRequest(ctx, "GET", url)
		req.Header = make(map[string][]string)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("authorization", fmt.Sprintf("Bearer %s", m.cfg.CDSWApiKey))
		resp, err := m.connections.HttpClient.Do(req)
		if err != nil {
			log.Printf("failed to fetch runs for experiment %s: %s", experimentId, err)
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Printf("failed to fetch runs: %s", resp.Status)
			return nil, fmt.Errorf("failed to fetch runs for experiment %s: %s", experimentId, resp.Status)
		}

		respBody, ioerr := io.ReadAll(resp.Body)
		if ioerr != nil {
			log.Printf("failed to read body: %s", ioerr)
			return nil, err
		}
		var runsResponse RunsResponse
		serr := json.Unmarshal(respBody, &runsResponse)
		if serr != nil {
			log.Printf("failed to unmarshal body: %s", serr)
			return nil, serr
		}
		for _, run := range runsResponse.Runs {
			runs = append(runs, &run)
		}
		if runsResponse.NextPageToken == "" {
			done = true
		} else {
			token = runsResponse.NextPageToken
		}
	}
	return runs, nil
}

func (m *PlatformMLFlow) CreateRun(ctx context.Context, experimentId string, name string, createdTs time.Time, tags []RunTag) (string, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s/runs", m.baseUrl, m.cfg.CDSWProjectID, experimentId)
	req := cbhttp.NewRequest(ctx, "POST", url)
	body := map[string]interface{}{
		"project_id":    m.cfg.CDSWProjectID,
		"experiment_id": experimentId,
		"start_time":    createdTs,
		"tags":          tags,
	}
	encoded, jerr := json.Marshal(body)
	if jerr != nil {
		log.Printf("failed to encode body: %s", jerr)
		return "", jerr
	}
	req.Body = io.NopCloser(bytes.NewReader(encoded))
	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", m.cfg.CDSWApiKey))
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Printf("failed to create experiment %s: %s", name, err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("failed to fetch experiments: %s", resp.Status)
		return "", fmt.Errorf("failed to create experiment %s: %s", name, resp.Status)
	}
	var run PlatformRun
	respBody, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		log.Printf("failed to read body: %s", ioerr)
		return "", ioerr
	}
	serr := json.Unmarshal(respBody, &run)
	if serr != nil {
		log.Printf("failed to unmarshal body: %s", serr)
		return "", serr
	}
	return run.Id, nil
}

func (m *PlatformMLFlow) CreateExperiment(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments", m.baseUrl, m.cfg.CDSWProjectID)
	req := cbhttp.NewRequest(ctx, "POST", url)
	body := map[string]interface{}{
		"project_id": m.cfg.CDSWProjectID,
		"name":       name,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		log.Printf("failed to encode body: %s", err)
		return "", err
	}
	req.Body = io.NopCloser(bytes.NewReader(encoded))
	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", m.cfg.CDSWApiKey))
	resp, lerr := m.connections.HttpClient.Do(req)
	if lerr != nil {
		if lerr.Code == 409 {
			experiment, gerr := m.GetExperimentByName(ctx, name)
			if gerr != nil {
				log.Printf("failed to fetch experiment %s: %s", name, gerr)
				return "", gerr
			}
			return experiment.ExperimentId, nil
		}
		log.Printf("failed to create experiment %s: %s", name, lerr)
		return "", lerr
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Printf("failed to fetch experiments: %s", resp.Status)
		return "", fmt.Errorf("failed to create experiment %s: %s", name, resp.Status)
	}

	respBody, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		log.Printf("failed to read body: %s", ioerr)
		return "", ioerr
	}
	var experiment PlatformExperiment
	serr := json.Unmarshal(respBody, &experiment)
	if serr != nil {
		log.Printf("failed to unmarshal body: %s", serr)
		return "", serr
	}
	return experiment.Id, nil
}

func (m *PlatformMLFlow) ListExperiments(ctx context.Context, maxItems int64, pageToken string) ([]*Experiment, error) {
	token := pageToken
	done := false
	experiments := make([]*Experiment, 0)
	for {
		if done {
			break
		}
		url := fmt.Sprintf("%s/api/v2/projects/%s/experiments?page_size=%d&page_token=%s", m.baseUrl, m.cfg.CDSWProjectID, maxItems, token)
		req := cbhttp.NewRequest(ctx, "GET", url)
		req.Header = make(map[string][]string)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("authorization", fmt.Sprintf("Bearer %s", m.cfg.CDSWApiKey))
		resp, err := m.connections.HttpClient.Do(req)
		if err != nil {
			log.Printf("failed to fetch experiments: %s", err)
			done = true
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Printf("failed to fetch experiments: %s", resp.Status)
			done = true
			continue
		}

		respBody, ioerr := io.ReadAll(resp.Body)
		if ioerr != nil {
			log.Printf("failed to read body: %s", ioerr)
			done = true
			continue
		}
		var experimentsResponse PlatformExperimentListResponse
		serr := json.Unmarshal(respBody, &experimentsResponse)
		if serr != nil {
			log.Printf("failed to unmarshal body: %s", serr)
			done = true
			continue
		}
		for _, experiment := range experimentsResponse.Experiments {
			experiments = append(experiments, &Experiment{
				ExperimentId:     experiment.Id,
				Name:             experiment.Name,
				ArtifactLocation: experiment.ArtifactLocation,
				LifecycleStage:   experiment.LifecycleStage,
				LastUpdatedTime:  experiment.LastUpdatedTime,
				CreatedTime:      experiment.CreatedTime,
				Tags:             experiment.Tags,
			})
		}
		if experimentsResponse.NextPageToken == "" {
			done = true
		} else {
			token = experimentsResponse.NextPageToken
		}
	}
	return experiments, nil
}

func (m *PlatformMLFlow) GetExperimentByName(ctx context.Context, name string) (*Experiment, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments", m.baseUrl, m.cfg.CDSWProjectID) // TODO figure out search_filter parameter
	req := cbhttp.NewRequest(ctx, "GET", url)
	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", m.cfg.CDSWApiKey))
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Printf("failed to fetch experiment %s: %s", name, err)
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, nil
	}
	defer resp.Body.Close()

	body, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		return nil, err
	}
	var experimentListResponse PlatformExperimentListResponse
	jerr := json.Unmarshal(body, &experimentListResponse)
	if jerr != nil {
		return nil, err
	}
	for _, experiment := range experimentListResponse.Experiments {
		if experiment.Name == name {
			return &Experiment{
				ExperimentId:     experiment.Id,
				Name:             experiment.Name,
				ArtifactLocation: experiment.ArtifactLocation,
				LifecycleStage:   experiment.LifecycleStage,
				LastUpdatedTime:  experiment.LastUpdatedTime,
				CreatedTime:      experiment.CreatedTime,
				Tags:             experiment.Tags,
			}, nil
		}
	}
	return nil, nil
}

func (m *PlatformMLFlow) GetExperiment(ctx context.Context, experimentId string) (*Experiment, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s", m.baseUrl, m.cfg.CDSWProjectID, experimentId)
	req := cbhttp.NewRequest(ctx, "GET", url)
	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", fmt.Sprintf("Bearer %s", m.cfg.CDSWApiKey))
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Printf("failed to fetch experiment %s: %s", experimentId, err)
		return nil, err
	}
	if resp.StatusCode == 404 {
		return nil, nil
	}
	defer resp.Body.Close()

	body, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		return nil, err
	}
	var experimentResponse ExperimentResponse
	jerr := json.Unmarshal(body, &experimentResponse)
	if jerr != nil {
		return nil, err
	}
	experiment := experimentResponse.Experiment
	return &experiment, nil
}

func (m *PlatformMLFlow) Metrics(ctx context.Context, experimentId string, runId string) ([]Metric, error) {
	if runId == "" {
		return nil, fmt.Errorf("runId is required")
	}

	run, err := m.GetRun(ctx, experimentId, runId)
	if err != nil {
		log.Printf("failed to fetch run %s: %s", runId, err)
		return nil, err
	}
	return run.Data.Metrics, nil
}
