package postgres

import (
	"context"
	"database/sql"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/config"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
	"time"
)

type Experiments struct {
	db  *lsql.Instance
	cfg *config.Config
}

var _ db.ExperimentService = &Experiments{}

func NewExperiments(instance *lsql.Instance, cfg *config.Config) db.ExperimentService {
	return &Experiments{
		db:  instance,
		cfg: cfg,
	}
}

func (e *Experiments) CreateExperiment(ctx context.Context, experimentId string, name string, createdTs time.Time, updatedTs time.Time) (*db.Experiment, error) {
	query := `
	INSERT INTO experiments (project_id, experiment_id, name, created, created_ts, updated, updated_ts, deleted) 
	VALUES (?, ?, ?, true, ?, false, ?, false)
	RETURNING id
	`
	id, err := e.db.ExecAndReturnId(ctx, query, e.cfg.CDSWProjectID, experimentId, name, createdTs, updatedTs)
	if err != nil {
		return nil, err
	}
	return e.GetExperimentById(ctx, id)
}

func (e *Experiments) UpdateExperimentCreatedAndTimestamp(ctx context.Context, id int64, created bool, ts time.Time) error {
	query := `
	UPDATE experiments SET created=?, created_ts=?
	WHERE id = ?
	`
	res, err := e.db.ExecContext(ctx, query, created, ts, id)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		log.Printf("no rows affected for experiment %d", id)
	}
	return nil
}

func (e *Experiments) UpdateExperimentUpdatedAndTimestamp(ctx context.Context, id int64, updated bool, ts time.Time) error {
	query := `
	UPDATE experiments SET updated=?, updated_ts=?
	WHERE id = ?
	`
	res, err := e.db.ExecContext(ctx, query, updated, ts, id)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		log.Printf("no rows affected for experiment %d", id)
	}
	return nil
}

func (e *Experiments) GetExperimentById(ctx context.Context, id int64) (*db.Experiment, error) {
	query := `
	SELECT id, name, experiment_id, created, updated, deleted, created_ts, updated_ts
	FROM experiments
	WHERE id = ? AND project_id = ?
	`
	row := e.db.QueryRowContext(ctx, query, id, e.cfg.CDSWProjectID)

	experiment, err := e.experimentFromRow(row)
	if err != nil {
		return nil, err
	}
	return experiment, nil
}

func (e *Experiments) GetExperimentByExperimentId(ctx context.Context, experimentId string) (*db.Experiment, error) {
	query := `
	SELECT id, name, experiment_id, created, updated, deleted, created_ts, updated_ts
	FROM experiments
	WHERE experiment_id = ? AND project_id = ?
	`
	row := e.db.QueryRowContext(ctx, query, experimentId, e.cfg.CDSWProjectID)

	experiment, err := e.experimentFromRow(row)
	if err != nil {
		return nil, err
	}
	return experiment, nil
}

func (e *Experiments) experimentFromRow(row lsql.RowScanner) (*db.Experiment, error) {
	experiment := &db.Experiment{}
	name := sql.NullString{}
	if err := row.Scan(&experiment.Id, &name, &experiment.ExperimentId, &experiment.Created, &experiment.Updated, &experiment.Deleted, &experiment.CreatedTs, &experiment.UpdatedTs); err != nil {
		return nil, err
	}
	if name.Valid {
		experiment.Name = name.String
	}
	return experiment, nil
}

func (e *Experiments) MarkExperimentIDForReconciliation(ctx context.Context, id int64) error {
	query := `
	UPDATE experiments SET updated=?, updated_ts=?
	WHERE id = ?
	`
	res, err := e.db.ExecContext(ctx, query, true, time.Now(), id)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		log.Printf("no rows affected for experiment %d", id)
	}
	return nil
}

func (e *Experiments) ListExperimentIDsForReconciliation(ctx context.Context, maxItems int64) ([]int64, error) {
	query := `
	SELECT id
	FROM experiments
	WHERE created = true OR updated = true AND project_id = ?
	`
	rows, err := e.db.QueryContext(ctx, query, e.cfg.CDSWProjectID)

	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (e *Experiments) ListExperiments(ctx context.Context) ([]*db.Experiment, error) {
	query := `
	SELECT id, name, experiment_id, created, updated, deleted, created_ts, updated_ts
	FROM experiments
	WHERE project_id = ?
	`
	rows, err := e.db.QueryContext(ctx, query, e.cfg.CDSWProjectID)

	if err != nil {
		return nil, err
	}
	response := make([]*db.Experiment, 0)
	for rows.Next() {
		experiment, err := e.experimentFromRow(rows)
		if err != nil {
			return nil, err
		}
		response = append(response, experiment)
	}

	return response, nil
}
