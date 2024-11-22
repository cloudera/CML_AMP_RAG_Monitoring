package datasource

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase"
	cbhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/clientbase/http"
	"io"
)

type RunData struct {
	Metrics []Metric `json:"metrics"`
}

type Run struct {
	Data RunData `json:"data"`
}

type RunResponse struct {
	Run Run `json:"run"`
}

type ArtifactsResponse struct {
	RootUri       string     `json:"root_uri"`
	Files         []Artifact `json:"files"`
	NextPageToken string     `json:"next_page_token"`
}

type ExperimentResponse struct {
	Experiment Experiment `json:"experiment"`
}

type MLFlow struct {
	baseUrl     string
	connections *clientbase.Connections
}

var _ DataStore = &MLFlow{}

func NewMLFlow(cfg *Config, connections *clientbase.Connections) DataStore {
	return &MLFlow{
		baseUrl:     cfg.MLFlowBaseUrl,
		connections: connections,
	}
}

func (m *MLFlow) Metrics(ctx context.Context, runId string) ([]Metric, error) {
	if runId == "" {
		return nil, fmt.Errorf("runId is required")
	}
	url := fmt.Sprintf("%s/api/2.0/mlflow/runs/get?run_id=%s", m.baseUrl, runId)
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
	url := fmt.Sprintf("%s/api/2.0/mlflow/artifacts/list?run_id=%s", m.baseUrl, runId)
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

func (m *MLFlow) ListExperiments(ctx context.Context) ([]*Experiment, error) {
	//TODO implement me
	panic("implement me")
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
