package postgres

import (
	"context"
	"database/sql"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/config"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
	"time"
)

type ExperimentRuns struct {
	db  *lsql.Instance
	cfg *config.Config
}

var _ db.ExperimentRunService = &ExperimentRuns{}

func NewExperimentRuns(instance *lsql.Instance, cfg *config.Config) db.ExperimentRunService {
	return &ExperimentRuns{
		db:  instance,
		cfg: cfg,
	}
}

func (e *ExperimentRuns) CreateExperimentRun(ctx context.Context, run *db.ExperimentRun) (*db.ExperimentRun, error) {
	query := `
	INSERT INTO experiment_runs (project_id, experiment_id, run_id, updated)
	VALUES (?, ?, ?, true)
	RETURNING id
	`
	args := []interface{}{e.cfg.CDSWProjectID, run.ExperimentId, run.RunId}
	id, err := e.db.ExecAndReturnId(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return &db.ExperimentRun{
		Id:           id,
		ExperimentId: run.ExperimentId,
		RunId:        run.RunId,
	}, nil
}

func (e *ExperimentRuns) GetExperimentRunById(ctx context.Context, id int64) (*db.ExperimentRun, error) {
	query := `
	SELECT id, experiment_id, run_id, created, updated, deleted, created_ts, updated_ts
	FROM experiment_runs
	WHERE id = ? AND project_id = ?
	`
	row := e.db.QueryRowContext(ctx, query, id, e.cfg.CDSWProjectID)

	if response, err := ExperimentRunInstance(row); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

func (e *ExperimentRuns) GetExperimentRun(ctx context.Context, experimentId string, runId string) (*db.ExperimentRun, error) {
	query := `
	SELECT id, experiment_id, run_id, created, updated, deleted, created_ts, updated_ts
	FROM experiment_runs
	WHERE experiment_id = ? AND run_id = ? AND project_id = ?
	`
	row := e.db.QueryRowContext(ctx, query, experimentId, runId, e.cfg.CDSWProjectID)

	if response, err := ExperimentRunInstance(row); err != nil {
		return nil, err
	} else {
		return response, nil
	}
}

func (e *ExperimentRuns) ListExperimentRuns(ctx context.Context, experimentId string) ([]*db.ExperimentRun, error) {
	query := `
	SELECT id, experiment_id, run_id, created, updated, deleted, created_ts, updated_ts
	FROM experiment_runs
	WHERE experiment_id = ? AND project_id = ?
	`
	args := []interface{}{experimentId, e.cfg.CDSWProjectID}
	rows, err := e.db.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, err
	}
	response := make([]*db.ExperimentRun, 0)
	for rows.Next() {
		if run, err := ExperimentRunInstance(rows); err != nil {
			return nil, err
		} else {
			response = append(response, run)
		}
	}

	return response, nil
}

func (e *ExperimentRuns) ListExperimentRunIdsForReconciliation(ctx context.Context, maxItems int64) ([]int64, error) {
	query := `
	SELECT id
	FROM experiment_runs
	WHERE deleted = false AND updated = true AND project_id = ?
	LIMIT ?
	`

	args := []interface{}{e.cfg.CDSWProjectID, maxItems}

	rows, err := e.db.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, err
	}
	response := make([]int64, 0)
	for rows.Next() {
		id := sql.NullInt64{}
		if err := rows.Scan(&id); err != nil {
			return nil, err
		} else {
			if !id.Valid {
				continue
			}
			response = append(response, id.Int64)
		}
	}

	return response, nil
}

func (e *ExperimentRuns) ListExperimentRunIdsForMetricReconciliation(ctx context.Context, maxItems int64) ([]int64, error) {
	query := `
	SELECT id
	FROM experiment_runs
	WHERE deleted = false AND reconcile_metrics = true AND project_id = ?
	LIMIT ?
	`

	args := []interface{}{e.cfg.CDSWProjectID, maxItems}

	rows, err := e.db.QueryContext(ctx, query, args...)

	if err != nil {
		return nil, err
	}
	response := make([]int64, 0)
	for rows.Next() {
		id := sql.NullInt64{}
		if err := rows.Scan(&id); err != nil {
			return nil, err
		} else {
			if !id.Valid {
				continue
			}
			response = append(response, id.Int64)
		}
	}

	return response, nil
}

func (e *ExperimentRuns) UpdateExperimentRunReconcileMetrics(ctx context.Context, id int64, reconcileMetrics bool) error {
	query := `
	UPDATE experiment_runs
	SET reconcile_metrics = ?
	WHERE id = ? 
	`
	args := []interface{}{reconcileMetrics, id}
	_, err := e.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (e *ExperimentRuns) UpdateExperimentRunUpdatedAndTimestamp(ctx context.Context, id int64, updated bool, updatedAt time.Time) error {
	query := `
	UPDATE experiment_runs
	SET updated = ?, updated_ts = ?
	WHERE id = ? 
	`
	args := []interface{}{updated, updatedAt, id}
	_, err := e.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	return nil
}

func (e *ExperimentRuns) DeleteExperimentRun(ctx context.Context, experimentId string, runId string) error {
	query := `
	DELETE FROM experiment_runs
	WHERE experiment_id = ? AND run_id = ?
	`
	args := []interface{}{experimentId, runId}
	_, err := e.db.ExecContext(ctx, query, args...)
	return err
}

func ExperimentRunInstance(scanner lsql.RowScanner) (*db.ExperimentRun, error) {
	run := &db.ExperimentRun{}
	if err := scanner.Scan(&run.Id, &run.ExperimentId, &run.RunId, &run.Created, &run.Updated, &run.Deleted, &run.CreatedTs, &run.UpdatedTs); err != nil {
		return nil, err
	}
	return run, nil
}
