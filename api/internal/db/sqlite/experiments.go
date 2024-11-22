package sqlite

import (
	"context"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
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

func (e *Experiments) ListExperiments(ctx context.Context) ([]*db.Experiment, error) {
	query := `
	SELECT id, experiment_id
	FROM experiment_runs
	`
	rows, err := e.db.QueryContext(ctx, query)

	if err != nil {
		return nil, err
	}
	response := make([]*db.Experiment, 0)
	for rows.Next() {
		experiment := &db.Experiment{}
		if err := rows.Scan(&experiment.Id, &experiment.ExperimentId); err != nil {
			return nil, err
		}
		response = append(response, experiment)
	}

	return response, nil
}
