package datasource

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase"
	cbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase/http"
	"io"
	"time"
)

// TODO: Remove this entire file once we are certain none of it is getting used

type MLFlow struct {
	baseUrl     string
	cfg         *Config
	connections *clientbase.Connections
}

var _ DataStore = &MLFlow{}

func NewMLFlow(baseUrl string, cfg *Config, connections *clientbase.Connections) DataStore {
	return &MLFlow{
		baseUrl:     baseUrl,
		cfg:         cfg,
		connections: connections,
	}
}

func (m *MLFlow) UpdateRun(ctx context.Context, run *Run) (*Run, error) {
	panic("local mlflow UpdateRun is not supported")
}

func (m *MLFlow) GetRun(ctx context.Context, experimentId string, runId string) (*Run, error) {
	url := fmt.Sprintf("%s/api/2.0/mlflow/runs/get?run_id=%s", m.baseUrl, runId)
	req := cbhttp.NewRequest(ctx, "GET", url)
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Debugf("failed to fetch run %s: %s", runId, err)
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

	var runResponse RunResponse
	jerr := json.Unmarshal(body, &runResponse)
	if jerr != nil {
		return nil, err
	}
	return &runResponse.Run, nil
}

func (m *MLFlow) ListRuns(ctx context.Context, experimentId string) ([]*Run, error) {
	token := ""
	done := false
	runs := make([]*Run, 0)
	for {
		if done {
			break
		}

		url := fmt.Sprintf("%s/api/2.0/mlflow/runs/search", m.baseUrl)
		req := cbhttp.NewRequest(ctx, "POST", url)
		body := map[string]interface{}{
			"experiment_ids": []string{experimentId},
			"page_token":     token,
		}
		encoded, serr := json.Marshal(body)
		if serr != nil {
			log.Debugf("failed to encode body: %s", serr)
			return nil, serr
		}
		req.Body = io.NopCloser(bytes.NewReader(encoded))
		req.Header = make(map[string][]string)
		req.Header.Set("Content-Type", "application/json")
		resp, lerr := m.connections.HttpClient.Do(req)
		if lerr != nil {
			log.Debugf("failed to fetch runs for experiment %s: %s", experimentId, lerr)
			return nil, lerr
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			log.Debugf("failed to fetch runs: %s", resp.Status)
			return nil, fmt.Errorf("failed to fetch runs for experiment %s: %s", experimentId, resp.Status)
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Debugf("failed to read body: %s", err)
			return nil, err
		}
		var runsResponse RunsResponse
		err = json.Unmarshal(respBody, &runsResponse)
		if err != nil {
			log.Debugf("failed to unmarshal body: %s", err)
			return nil, err
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

func (m *MLFlow) CreateRun(ctx context.Context, experimentId string, name string, createdTs time.Time, tags []RunTag) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (m *MLFlow) Metrics(ctx context.Context, experimentId string, runId string) ([]Metric, error) {
	if runId == "" {
		return nil, fmt.Errorf("runId is required")
	}
	url := fmt.Sprintf("%s/api/2.0/mlflow/runs/get?run_id=%s", m.baseUrl, runId)

	req := cbhttp.NewRequest(ctx, "GET", url)
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Debugf("failed to fetch metrics for run %s: %s", runId, err)
		return nil, err
	}
	if resp.StatusCode == 404 {
		log.Debugf("metrics not found for run %s", runId)
		return []Metric{}, nil
	}
	defer resp.Body.Close()

	body, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		return nil, err
	}

	var runResponse RunResponse
	jerr := json.Unmarshal(body, &runResponse)
	if jerr != nil {
		return nil, err
	}
	return runResponse.Run.Data.Metrics, nil
}

func (m *MLFlow) Artifacts(ctx context.Context, runId string, path *string) ([]Artifact, error) {
	if runId == "" {
		return nil, fmt.Errorf("runId is required")
	}
	url := fmt.Sprintf("%s/api/2.0/mlflow/artifacts/list?run_id=%s", m.baseUrl, runId)
	if path != nil {
		url = fmt.Sprintf("%s&path=%s", url, *path)
	}
	log.Debugf("fetching artifacts for run %s using url %s", runId, url)
	req := cbhttp.NewRequest(ctx, "GET", url)
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		if err.Code == 404 {
			log.Debugf("run %s has no artifacts", runId)
			return []Artifact{}, nil
		}
		log.Debugf("failed to fetch artifacts for run %s: %s", runId, err)
		return nil, err
	}
	if resp.StatusCode == 404 {
		return []Artifact{}, nil
	}
	defer resp.Body.Close()

	body, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		return nil, err
	}

	var artifactsResponse ArtifactsResponse
	jerr := json.Unmarshal(body, &artifactsResponse)
	if jerr != nil {
		return nil, err
	}
	return artifactsResponse.Files, nil
}

func (m *MLFlow) GetArtifact(ctx context.Context, runId string, path string) ([]byte, error) {
	if runId == "" {
		return nil, fmt.Errorf("runId is required")
	}
	url := fmt.Sprintf("%s/get-artifact?run_id=%s&path=%s", m.baseUrl, runId, path)
	req := cbhttp.NewRequest(ctx, "GET", url)
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Debugf("failed to fetch arrtifacts for run %s: %s", runId, err)
		return nil, err
	}
	if resp.StatusCode == 404 {
		return []byte{}, nil
	}
	defer resp.Body.Close()

	body, ioerr := io.ReadAll(resp.Body)
	if ioerr != nil {
		return nil, err
	}

	return body, nil
}

func (m *MLFlow) UploadArtifact(ctx context.Context, experimentId string, runId string, path string, data []byte) (string, error) {
	panic("upload to local mlflow not supported")
}

func (m *MLFlow) CreateExperiment(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments", m.baseUrl, m.cfg.CDSWProjectID)
	req := cbhttp.NewRequest(ctx, "POST", url)
	body := map[string]interface{}{
		"project_id": m.cfg.CDSWProjectID,
		"name":       name,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		log.Debugf("failed to encode body: %s", err)
		return "", err
	}
	req.Body = io.NopCloser(bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Debugf("failed to create experiment %s: %s", name, err)
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Debugf("failed to fetch experiments: %s", resp.Status)
		return "", fmt.Errorf("failed to create experiment %s: %s", name, resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Debugf("failed to read body: %s", err)
		return "", err
	}
	var experiment Experiment
	err = json.Unmarshal(respBody, &experiment)
	if err != nil {
		log.Debugf("failed to unmarshal body: %s", err)
		return "", err
	}
	return experiment.ExperimentId, nil
}

func (m *MLFlow) ListExperiments(ctx context.Context, maxItems int64, pageToken string) ([]*Experiment, error) {
	url := fmt.Sprintf("%s/api/2.0/mlflow/experiments/search", m.baseUrl)
	done := false
	token := pageToken
	experiments := make([]*Experiment, 0)
	for {
		if done {
			break
		}
		body := map[string]interface{}{
			"max_results": 1000,
			"page_token":  token,
		}

		encoded, err := json.Marshal(body)
		if err != nil {
			log.Debugf("failed to encode body: %s", err)
		}
		req := cbhttp.NewRequest(ctx, "POST", url)
		req.Body = io.NopCloser(bytes.NewReader(encoded))
		req.Header = make(map[string][]string)
		req.Header.Add("Content-Type", "application/json")
		resp, lerr := m.connections.HttpClient.Do(req)

		if lerr != nil {
			if lerr.Code != 404 {
				log.Debugf("failed to fetch local experiments: %s", lerr)
			}
			done = true
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			if resp.StatusCode != 404 {
				log.Debugf("failed to fetch experiments: %s", err)
			}
			done = true
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Debugf("failed to read body: %s", err)
			done = true
			continue
		}
		var experimentsResponse ExperimentListResponse
		err = json.Unmarshal(respBody, &experimentsResponse)
		if err != nil {
			log.Debugf("failed to unmarshal body: %s", err)
			done = true
			continue
		}
		for _, experiment := range experimentsResponse.Experiments {
			experiments = append(experiments, experiment)
		}
		if experimentsResponse.NextPageToken == "" {
			done = true
		} else {
			token = experimentsResponse.NextPageToken
		}
	}
	return experiments, nil
}

func (m *MLFlow) GetExperiment(ctx context.Context, experimentId string) (*Experiment, error) {
	url := fmt.Sprintf("%s/api/2.0/mlflow/experiments/get?experiment_id=%s", m.baseUrl, experimentId)
	req := cbhttp.NewRequest(ctx, "GET", url)
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
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
