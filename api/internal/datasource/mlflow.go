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

func (m *MLFlow) UpdateRun(ctx context.Context, run *Run) error {
	//TODO implement me
	panic("implement me")
}

func (m *MLFlow) GetRun(ctx context.Context, experimentId string, runId string) (*Run, error) {
	url := fmt.Sprintf("%s/api/2.0/mlflow/get?run_id=%s", m.baseUrl, runId)
	req := cbhttp.NewRequest(ctx, "GET", url)
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
		encoded, err := json.Marshal(body)
		if err != nil {
			log.Printf("failed to encode body: %s", err)
			return nil, err
		}
		req.Body = io.NopCloser(bytes.NewReader(encoded))
		req.Header.Set("Content-Type", "application/json")
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

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("failed to read body: %s", err)
			return nil, err
		}
		var runsResponse RunsResponse
		err = json.Unmarshal(respBody, &runsResponse)
		if err != nil {
			log.Printf("failed to unmarshal body: %s", err)
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

func (m *MLFlow) Metrics(ctx context.Context, runId string) ([]Metric, error) {
	if runId == "" {
		return nil, fmt.Errorf("runId is required")
	}
	url := fmt.Sprintf("%s/api/2.0/runs/runs/get?run_id=%s", m.baseUrl, runId)
	req := cbhttp.NewRequest(ctx, "GET", url)
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Printf("failed to fetch metrics for run %s: %s", runId, err)
		return nil, err
	}
	if resp.StatusCode == 404 {
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
	url := fmt.Sprintf("%s/api/2.0/runs/artifacts/list?run_id=%s", m.baseUrl, runId)
	if path != nil {
		url = fmt.Sprintf("%s&path=%s", url, *path)
	}
	req := cbhttp.NewRequest(ctx, "GET", url)
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Printf("failed to fetch arrtifacts for run %s: %s", runId, err)
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
		log.Printf("failed to fetch arrtifacts for run %s: %s", runId, err)
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

func (m *MLFlow) CreateExperiment(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments", m.baseUrl, m.cfg.CDSWProjectNum)
	req := cbhttp.NewRequest(ctx, "POST", url)
	body := map[string]interface{}{
		"project_id": m.cfg.CDSWProjectNum,
		"name":       name,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		log.Printf("failed to encode body: %s", err)
		return "", err
	}
	req.Body = io.NopCloser(bytes.NewReader(encoded))
	req.Header.Set("Content-Type", "application/json")
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read body: %s", err)
		return "", err
	}
	var experiment Experiment
	err = json.Unmarshal(respBody, &experiment)
	if err != nil {
		log.Printf("failed to unmarshal body: %s", err)
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
			log.Printf("failed to encode body: %s", err)
		}
		req := cbhttp.NewRequest(ctx, "POST", url)
		req.Body = io.NopCloser(bytes.NewReader(encoded))
		req.Header = make(map[string][]string)
		req.Header.Add("Content-Type", "application/json")
		resp, lerr := m.connections.HttpClient.Do(req)

		if lerr != nil {
			if lerr.Code != 404 {
				log.Errorf("failed to fetch local experiments: %s", lerr)
			}
			done = true
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			if resp.StatusCode != 404 {
				log.Printf("failed to fetch experiments: %s", err)
			}
			done = true
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("failed to read body: %s", err)
			done = true
			continue
		}
		var experimentsResponse ExperimentListResponse
		err = json.Unmarshal(respBody, &experimentsResponse)
		if err != nil {
			log.Printf("failed to unmarshal body: %s", err)
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
