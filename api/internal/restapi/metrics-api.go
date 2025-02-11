package restapi

import (
	"context"
	"github.com/go-openapi/strfmt"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/models"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi/operations/metrics"
)

var _ restapi.MetricsAPI = &MetricsAPI{}

type MetricsAPI struct {
	db db.Database
}

func NewMetricsAPI(db db.Database) *MetricsAPI {
	return &MetricsAPI{
		db: db,
	}
}

func (m MetricsAPI) PostMetrics(ctx context.Context, params metrics.PostMetricsParams) (*metrics.PostMetricsOK, *lhttp.HttpError) {
	if params.Body == nil {
		return nil, lhttp.NewBadRequest("body is required")
	}
	if params.Body.Metrics == nil || len(params.Body.Metrics) == 0 {
		return nil, lhttp.NewBadRequest("body.metrics is required")
	}
	for _, metric := range params.Body.Metrics {
		if metric.ExperimentID == "" {
			return nil, lhttp.NewBadRequest("experiment_id is required")
		}
		if metric.ExperimentRunID == "" {
			return nil, lhttp.NewBadRequest("experiment_run_id is required")
		}
		if metric.Name == "" {
			return nil, lhttp.NewBadRequest("metric name is required")
		}
		tags := make(map[string]string)
		for _, tag := range metric.Tags {
			tags[tag.Key] = tag.Value
		}
		newMetric := &db.Metric{
			ExperimentId: metric.ExperimentID,
			RunId:        metric.ExperimentRunID,
			Name:         metric.Name,
			Tags:         tags,
		}
		if metric.Value == nil {
			log.Printf("Metric %s has no value", metric.Name)
			continue
		}
		if metric.Value.MetricType == string(db.MetricTypeNumeric) {
			newMetric.Type = db.MetricTypeNumeric
			newMetric.ValueNumeric = &metric.Value.NumericValue
		} else {
			newMetric.Type = db.MetricTypeText
			newMetric.ValueText = &metric.Value.StringValue
		}

		result, err := m.db.Metrics().CreateMetric(ctx, newMetric)
		if err != nil {
			return nil, lhttp.NewInternalError(err.Error())
		}
		log.Printf("Created metric: %d", result.Id)
	}

	return &metrics.PostMetricsOK{}, nil
}

func (m MetricsAPI) GetMetricsNames(ctx context.Context, params metrics.GetMetricsNamesParams) (*metrics.GetMetricsNamesOK, *lhttp.HttpError) {
	if params.ExperimentID == nil || *params.ExperimentID == "" {
		return nil, lhttp.NewBadRequest("experiment_id is required")
	}
	results, err := m.db.Metrics().ListMetricNames(ctx, params.ExperimentID)
	if err != nil {
		return nil, lhttp.NewInternalError(err.Error())
	}
	return &metrics.GetMetricsNamesOK{
		Payload: results,
	}, nil
}

func (m MetricsAPI) PostMetricsList(ctx context.Context, params metrics.PostMetricsListParams) (*metrics.PostMetricsListOK, *lhttp.HttpError) {
	if params.Body == nil {
		return nil, lhttp.NewBadRequest("body is required")
	}
	results, err := m.db.Metrics().ListMetrics(ctx, &params.Body.ExperimentID, params.Body.RunIds, params.Body.MetricNames)
	if err != nil {
		return nil, lhttp.NewInternalError(err.Error())
	}
	payload := make([]*models.Metric, 0)

	for _, metric := range results {
		tags := make([]*models.MetricTag, 0)
		for k, v := range metric.Tags {
			tags = append(tags, &models.MetricTag{
				Key:   k,
				Value: v,
			})
		}
		result := &models.Metric{
			ExperimentID:    metric.ExperimentId,
			ExperimentRunID: metric.RunId,
			ID:              metric.Id,
			Name:            metric.Name,
			Tags:            tags,
			Ts:              strfmt.DateTime(*metric.Timestamp),
		}
		if metric.Type == db.MetricTypeNumeric {
			result.Value = &models.MetricValue{
				MetricType:   string(db.MetricTypeNumeric),
				NumericValue: *metric.ValueNumeric,
			}
		} else {
			result.Value = &models.MetricValue{
				MetricType:  string(db.MetricTypeText),
				StringValue: *metric.ValueText,
			}
		}
		payload = append(payload, result)
	}
	return &metrics.PostMetricsListOK{
		Payload: payload,
	}, nil
}

func (m MetricsAPI) Shutdown() error {
	return nil
}
