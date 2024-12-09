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

func (m *PlatformMLFlow) UpdateRun(ctx context.Context, run *Run) error {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s/runs/%s", m.baseUrl, m.cfg.RemoteProjectID, run.Info.ExperimentId, run.Info.RunId)
	req := cbhttp.NewRequest(ctx, "POST", url)

	encoded, err := json.Marshal(run)
	if err != nil {
		log.Printf("failed to encode body: %s", err)
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(encoded))
	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
	resp, err := m.connections.HttpClient.Do(req)
	if err != nil {
		log.Printf("failed to update run %s: %s", run.Info.RunId, err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to update run %s: %s", run.Info.RunId, resp.Status)
	}
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return nil
}

func (m *PlatformMLFlow) GetRun(ctx context.Context, experimentId string, runId string) (*Run, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s/runs/%s", m.baseUrl, m.cfg.RemoteProjectID, experimentId, runId)
	req := cbhttp.NewRequest(ctx, "GET", url)

	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
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

	var run Run
	jerr := json.Unmarshal(body, &run)
	if jerr != nil {
		return nil, err
	}
	return &run, nil
}

func (m *PlatformMLFlow) ListRuns(ctx context.Context, experimentId string) ([]*Run, error) {
	token := ""
	done := false
	runs := make([]*Run, 0)
	for {
		if done {
			break
		}
		url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s/runs?page_token=%s", m.baseUrl, m.cfg.RemoteProjectID, experimentId, token)
		req := cbhttp.NewRequest(ctx, "GET", url)
		req.Header = make(map[string][]string)
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
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments/%s/runs", m.baseUrl, m.cfg.RemoteProjectID, experimentId)
	req := cbhttp.NewRequest(ctx, "POST", url)
	body := map[string]interface{}{
		"project_id":    m.cfg.RemoteProjectID,
		"experiment_id": experimentId,
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		log.Printf("failed to encode body: %s", err)
		return "", err
	}
	req.Body = io.NopCloser(bytes.NewReader(encoded))
	req.Header = make(map[string][]string)
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
	var run Run
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("failed to read body: %s", err)
		return "", err
	}
	err = json.Unmarshal(respBody, &run)
	if err != nil {
		log.Printf("failed to unmarshal body: %s", err)
		return "", err
	}
	return run.Info.RunId, nil
}

func (m *PlatformMLFlow) CreateExperiment(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf("%s/api/v2/projects/%s/experiments", m.baseUrl, m.cfg.RemoteProjectID)
	req := cbhttp.NewRequest(ctx, "POST", url)
	body := map[string]interface{}{
		"project_id": m.cfg.RemoteProjectID,
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

func (m *PlatformMLFlow) ListExperiments(ctx context.Context, maxItems int64, pageToken string) ([]*Experiment, error) {
	token := pageToken
	done := false
	experiments := make([]*Experiment, 0)
	for {
		if done {
			break
		}
		url := fmt.Sprintf("%s/api/v2/experiments?page_size=%d&page_token=%s", m.baseUrl, maxItems, token)
		req := cbhttp.NewRequest(ctx, "GET", url)
		req.Header = make(map[string][]string)
		req.Header.Set("Content-Type", "application/json")
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
		var experimentsResponse ExperimentListResponse
		serr := json.Unmarshal(respBody, &experimentsResponse)
		if serr != nil {
			log.Printf("failed to unmarshal body: %s", serr)
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

func (m *PlatformMLFlow) GetExperiment(ctx context.Context, experimentId string) (*Experiment, error) {
	url := fmt.Sprintf("%s/api/2.0/mlflow/experiments/get?experiment_id=%s", m.baseUrl, experimentId)
	req := cbhttp.NewRequest(ctx, "GET", url)
	req.Header = make(map[string][]string)
	req.Header.Set("Content-Type", "application/json")
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
