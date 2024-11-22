package db

import (
	"context"
	"time"
)

type MetricType string

const (
	MetricTypeNumeric MetricType = "numeric"
	MetricTypeText    MetricType = "text"
)

type Metric struct {
	Id           int64
	ExperimentId string
	RunId        string
	Name         string
	Type         MetricType
	ValueNumeric *float64
	ValueText    *string
	Tags         map[string]string
	Timestamp    *time.Time
}

type MetricsService interface {
	CreateMetric(ctx context.Context, m *Metric) (*Metric, error)
	GetMetric(ctx context.Context, id int64) (*Metric, error)
	ListMetrics(ctx context.Context, experimentId *string, runIds []string, metricNames []string) ([]*Metric, error)
}
