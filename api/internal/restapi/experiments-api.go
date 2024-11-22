package restapi

import (
	"context"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi/operations/experiments"
)

var _ restapi.ExperimentsAPI = &ExperimentAPI{}

type ExperimentAPI struct {
	db db.Database
}

func NewExperimentAPI(db db.Database) *ExperimentAPI {
	return &ExperimentAPI{db: db}
}

func (e *ExperimentAPI) GetExperiments(ctx context.Context, params experiments.GetExperimentsParams) (*experiments.GetExperimentsOK, *lhttp.HttpError) {
	result, err := e.db.Experiments().ListExperiments(ctx)
	if err != nil {
		return nil, lhttp.NewInternalError(err.Error())
	}
	payload := make([]string, 0)

	for _, experiment := range result {
		payload = append(payload, experiment.ExperimentId)
	}
	return &experiments.GetExperimentsOK{
		Payload: payload,
	}, nil
}

func (e *ExperimentAPI) Shutdown() error {
	return nil
}
