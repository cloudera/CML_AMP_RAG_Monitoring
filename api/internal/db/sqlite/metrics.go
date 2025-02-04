package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
	"strings"
	"time"
)

type Metrics struct {
	db *lsql.Instance
}

var _ db.MetricsService = &Metrics{}

func NewMetrics(instance *lsql.Instance) db.MetricsService {
	return &Metrics{
		db: instance,
	}
}

func (r *Metrics) CreateMetric(ctx context.Context, m *db.Metric) (*db.Metric, error) {
	existingMetrics, err := r.ListMetrics(ctx, &m.ExperimentId, []string{m.RunId}, []string{m.Name})
	if err != nil {
		return nil, err
	}
	if len(existingMetrics) > 0 {
		for _, existingMetric := range existingMetrics {
			tags := existingMetric.Tags
			if len(tags) != len(m.Tags) {
				continue
			}
			tagsMatch := false
			for k, v := range m.Tags {
				if tags[k] == v {
					tagsMatch = true
					break
				}
			}
			if tagsMatch {
				log.Printf("Metric %s already exists for experiment %s and run %s : %d", m.Name, m.ExperimentId, m.RunId, existingMetric.Id)
				return existingMetric, nil
			}
		}
	}

	query := `
	INSERT INTO metrics (experiment_id, run_id, name, value_numeric, value_text, tags, ts)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	ts := time.Now()
	if m.Timestamp != nil {
		ts = *m.Timestamp
	}

	tags, err := json.Marshal(m.Tags)
	if err != nil {
		return nil, err
	}

	args := []interface{}{m.ExperimentId, m.RunId, m.Name, m.ValueNumeric, m.ValueText, tags, ts}
	id, err := r.db.ExecAndReturnId(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &db.Metric{
		Id:           id,
		ExperimentId: m.ExperimentId,
		RunId:        m.RunId,
		Name:         m.Name,
		ValueNumeric: m.ValueNumeric,
		ValueText:    m.ValueText,
		Timestamp:    &ts,
	}, nil
}

func (r *Metrics) GetMetric(ctx context.Context, id int64) (*db.Metric, error) {
	query := `
	SELECT id, experiment_id, run_id, name, value_numeric, value_text, tags, ts
	FROM metrics
	WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)

	if response, err := MetricInstance(row); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

func (r *Metrics) ListMetrics(ctx context.Context, experimentId *string, runIds []string, metricNames []string) ([]*db.Metric, error) {
	query := `
	SELECT id, experiment_id, run_id, name, value_numeric, value_text, tags, ts
	FROM metrics
	`
	conditions := []string{}
	parameters := []interface{}{}
	if experimentId != nil && *experimentId != "" {
		conditions = append(conditions, "experiment_id = ?")
		parameters = append(parameters, *experimentId)
	}
	if runIds != nil && len(runIds) != 0 {
		conditions = append(conditions, "run_id IN (?)")
		parameters = append(parameters, runIds)
	}
	if metricNames != nil && len(metricNames) != 0 {
		conditions = append(conditions, "name IN (?)")
		parameters = append(parameters, metricNames)
	}
	if len(conditions) > 0 {
		query = query + " WHERE " + strings.Join(conditions, " AND ")
	}
	query = query + " ORDER BY experiment_id, run_id, name, ts"
	rows, err := r.db.QueryContext(ctx, query, parameters...)

	if err != nil {
		return nil, err
	}
	response := make([]*db.Metric, 0)
	for rows.Next() {
		if metric, err := MetricInstance(rows); err != nil {
			return nil, err
		} else {
			response = append(response, metric)
		}
	}

	return response, nil
}

func MetricInstance(scanner lsql.RowScanner) (*db.Metric, error) {
	metric := &db.Metric{}
	numericValue := sql.NullFloat64{}
	textValue := sql.NullString{}
	ts := sql.NullTime{}
	tagsStr := sql.NullString{}
	err := scanner.Scan(&metric.Id, &metric.ExperimentId, &metric.RunId, &metric.Name, &numericValue, &textValue, &tagsStr, &ts)
	if err != nil {
		return nil, err
	}
	if ts.Valid {
		metric.Timestamp = &ts.Time
	}
	if metric.Tags == nil {
		metric.Tags = make(map[string]string)
	}
	if tagsStr.Valid {
		err = json.Unmarshal([]byte(tagsStr.String), &metric.Tags)
		if err != nil {
			return nil, err
		}
	}
	if numericValue.Valid {
		metric.Type = db.MetricTypeNumeric
		metric.ValueNumeric = &numericValue.Float64
	}
	if textValue.Valid {
		metric.Type = db.MetricTypeText
		metric.ValueText = &textValue.String
	}
	return metric, nil
}
