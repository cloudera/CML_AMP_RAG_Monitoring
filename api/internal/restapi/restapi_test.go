package restapi

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/models"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi/operations/experiments"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi/operations/metrics"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi/operations/runs"
	"pgregory.net/rapid"
	"testing"
)

func TestExperimentsAPI(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		experimentsMocks := &db.ExperimentsMock{
			Experiments: rapid.SliceOf(db.ExperimentGenerator()).Draw(t, "experiments"),
		}

		database := &db.SqliteDatabase{
			Experiments: experimentsMocks,
		}
		api := NewExperimentAPI(database)
		params := experiments.GetExperimentsParams{}

		ok, err := api.GetExperiments(context.TODO(), params)
		if err != nil {
			t.Fatalf("Error: %v", err)
		}
		assert.Equal(t, len(ok.Payload), len(experimentsMocks.Experiments))
	})
}

func TestExperimentRunsAPI(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		mockRuns := rapid.SliceOf(db.ExperimentRunGenerator()).Draw(t, "experiment_runs")
		var mocksByExperimentId = make(map[string][]*db.ExperimentRun)
		for _, run := range mockRuns {
			mocksByExperimentId[run.ExperimentId] = append(mocksByExperimentId[run.ExperimentId], run)
		}
		experimentsRunMocks := &db.ExperimentRunsMock{
			ExperimentRuns: mockRuns,
		}

		database := &db.SqliteDatabase{
			ExperimentRuns: experimentsRunMocks,
		}
		ctx := context.TODO()

		var experimentIds []string
		for _, run := range experimentsRunMocks.ExperimentRuns {
			experimentIds = append(experimentIds, run.ExperimentId)
		}

		var sample string
		if len(experimentIds) > 0 {
			sample = rapid.SampledFrom(experimentIds).Draw(t, "experiment_id")
		} else {
			sample = ""
		}
		api := NewExperimentRunsAPI(database)
		params := runs.PostRunsListParams{
			Body: &models.ExperimentRunListFilter{
				ExperimentID: sample,
			},
		}

		listOk, err := api.PostRunsList(ctx, params)
		if err != nil {
			if err.Code == 400 {
				assert.Equal(t, "", params.Body.ExperimentID)
			} else {
				t.Fatalf("unexpected error %v", err)
			}
		} else {
			// Property: The length of the payload should be equal to the length of the experiment runs with the same experiment ID
			assert.Equal(t, len(listOk.Payload), len(mocksByExperimentId[params.Body.ExperimentID]))
		}

		runParams := runs.PostRunsParams{
			Body: &models.ExperimentRun{
				ExperimentID:    rapid.StringMatching("[a-z0-9A-Z]{16}").Draw(t, "experiment_id"),
				ExperimentRunID: rapid.StringMatching("[a-z0-9A-Z]{16}").Draw(t, "run_id"),
			},
		}

		postOk, err := api.PostRuns(ctx, runParams)
		if err != nil {
			if err.Code == 400 {
				assert.Equal(t, "", runParams.Body.ExperimentID)
			} else {
				t.Fatalf("unexpected error %v", err)
			}
		} else {
			// Property: The experiment ID and experiment run ID should be the same as the ones in the request
			assert.Equal(t, postOk.Payload.ExperimentID, runParams.Body.ExperimentID)
			assert.Equal(t, postOk.Payload.ExperimentRunID, runParams.Body.ExperimentRunID)

			afterPostListOk, err := api.PostRunsList(ctx, runs.PostRunsListParams{
				Body: &models.ExperimentRunListFilter{
					ExperimentID: postOk.Payload.ExperimentID,
				},
			})
			assert.Nil(t, err)

			// Property: The length of the payload should be equal to the length of the experiment runs with the same experiment ID + 1
			assert.Equal(t, len(mocksByExperimentId[runParams.Body.ExperimentID])+1, len(afterPostListOk.Payload))

			deleteOk, err := api.DeleteRuns(ctx, runs.DeleteRunsParams{
				ExperimentID: &postOk.Payload.ExperimentID,
				RunID:        &postOk.Payload.ExperimentRunID,
			})
			assert.Nil(t, err)
			assert.NotNil(t, deleteOk)

			afterDeleteListOk, err := api.PostRunsList(ctx, runs.PostRunsListParams{
				Body: &models.ExperimentRunListFilter{
					ExperimentID: postOk.Payload.ExperimentID,
				},
			})
			assert.Nil(t, err)
			assert.Equal(t, len(afterPostListOk.Payload)-1, len(afterDeleteListOk.Payload))

		}
	})
}

func TestMetricsAPI(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		metricsMocks := &db.MetricsMock{
			CreatedMetrics: []*db.Metric{},
		}

		database := &db.SqliteDatabase{
			Metrics: metricsMocks,
		}
		api := NewMetricsAPI(database)

		postMetrics := rapid.SliceOf(db.MetricGenerator()).Draw(t, "metrics")

		ok, err := api.PostMetrics(context.TODO(), metrics.PostMetricsParams{
			Body: &models.Metrics{
				Metrics: postMetrics,
			},
		})

		if err != nil {
			if err.Code != 400 {
				t.Fatalf("Error: %v", err)
			}
			return
		}
		assert.NotNil(t, ok)

		// Property: Each element of postMetrics should now be in CreatedMetrics
		for _, metric := range postMetrics {
			found := false
			for _, createdMetric := range metricsMocks.CreatedMetrics {
				if metric.ExperimentID == createdMetric.ExperimentId &&
					metric.ExperimentRunID == createdMetric.RunId &&
					metric.Name == createdMetric.Name {
					found = true
					break
				}
			}
			assert.True(t, found)
		}

		metric := rapid.SampledFrom(postMetrics).Draw(t, "metric")
		listOk, err := api.PostMetricsList(context.TODO(), metrics.PostMetricsListParams{
			// TODO: try more combinations of metric names and run ids
			Body: &models.MetricListFilter{
				ExperimentID: metric.ExperimentID,
				MetricNames:  []string{metric.Name},
				RunIds:       []string{metric.ExperimentRunID},
			},
		})
		assert.Nil(t, err)
		assert.NotNil(t, listOk)
		assert.Equal(t, 1, len(listOk.Payload))
	})
}
