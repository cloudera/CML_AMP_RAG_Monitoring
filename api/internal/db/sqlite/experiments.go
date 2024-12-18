package sqlite

import (
	"context"
	"database/sql"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
	"time"
)

type Experiments struct {
	db *lsql.Instance
}

var _ db.ExperimentService = &Experiments{}

func NewExperiments(instance *lsql.Instance) db.ExperimentService {
	return &Experiments{
		db: instance,
	}
}

func (e *Experiments) CreateExperiment(ctx context.Context, experimentId string, createdTs time.Time, updatedTs time.Time) (*db.Experiment, error) {
	query := `
	INSERT INTO experiments (experiment_id, created, created_ts, updated, updated_ts, deleted) VALUES (?, true, ?, false, ?, false)
	`
	id, err := e.db.ExecAndReturnId(ctx, query, experimentId, createdTs, updatedTs)
	if err != nil {
		return nil, err
	}
	return e.GetExperimentById(ctx, id)
}

func (e *Experiments) UpdateRemoteExperimentId(ctx context.Context, id int64, remoteExperimentId string) error {
	query := `
	UPDATE experiments SET remote_experiment_id=?
	WHERE id = ?
	`
	res, err := e.db.ExecContext(ctx, query, remoteExperimentId, id)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		log.Printf("no rows affected for experiment %d", id)
	}
	return nil
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
	SELECT id, experiment_id, remote_experiment_id, created, updated, deleted, created_ts, updated_ts
	FROM experiments
	WHERE id = ?
	`
	row := e.db.QueryRowContext(ctx, query, id)

	experiment, err := e.experimentFromRow(row)
	if err != nil {
		return nil, err
	}
	return experiment, nil
}

func (e *Experiments) GetExperimentByExperimentId(ctx context.Context, experimentId string) (*db.Experiment, error) {
	query := `
	SELECT id, experiment_id, remote_experiment_id, created, updated, deleted, created_ts, updated_ts
	FROM experiments
	WHERE experiment_id = ?
	`
	row := e.db.QueryRowContext(ctx, query, experimentId)

	experiment, err := e.experimentFromRow(row)
	if err != nil {
		return nil, err
	}
	return experiment, nil
}

func (e *Experiments) experimentFromRow(row lsql.RowScanner) (*db.Experiment, error) {
	experiment := &db.Experiment{}
	remoteExperimentId := sql.NullString{}
	if err := row.Scan(&experiment.Id, &experiment.ExperimentId, &remoteExperimentId, &experiment.Created, &experiment.Updated, &experiment.Deleted, &experiment.CreatedTs, &experiment.UpdatedTs); err != nil {
		return nil, err
	}
	if remoteExperimentId.Valid {
		experiment.RemoteExperimentId = remoteExperimentId.String
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
	WHERE created = true OR updated = true
	`
	rows, err := e.db.QueryContext(ctx, query)

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
	SELECT id, experiment_id, remote_experiment_id, created, updated, deleted, created_ts, updated_ts
	FROM experiments
	`
	rows, err := e.db.QueryContext(ctx, query)

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
