package restapi

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/models"
	lhttp "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/http"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/restapi/operations/runs"
)

var _ restapi.RunsAPI = &ExperimentRunsAPI{}

type ExperimentRunsAPI struct {
	db        db.Database
	datastore datasource.DataStores
}

func NewExperimentRunsAPI(db db.Database, datastore datasource.DataStores) *ExperimentRunsAPI {
	return &ExperimentRunsAPI{db: db, datastore: datastore}
}

func (e ExperimentRunsAPI) PostRunsList(ctx context.Context, params runs.PostRunsListParams) (*runs.PostRunsListOK, *lhttp.HttpError) {
	if params.Body.ExperimentID == "" {
		return nil, lhttp.NewBadRequest("experiment_id is required")
	}
	ers, err := e.db.ExperimentRuns().ListExperimentRuns(ctx, params.Body.ExperimentID)
	if err != nil {
		return nil, lhttp.NewInternalError(err.Error())
	}
	payload := make([]*models.ExperimentRun, 0)
	for _, run := range ers {
		payload = append(payload, &models.ExperimentRun{
			ExperimentID:    run.ExperimentId,
			ExperimentRunID: run.RunId,
			ID:              run.Id,
		})
	}
	return &runs.PostRunsListOK{
		Payload: payload,
	}, nil
}

func (e ExperimentRunsAPI) GetRunsParameters(ctx context.Context, params runs.GetRunsParametersParams) (*runs.GetRunsParametersOK, *lhttp.HttpError) {
	if params.ExperimentID == nil || *params.ExperimentID == "" {
		return nil, lhttp.NewBadRequest("experiment_id is required")
	}
	if params.RunID == nil || *params.RunID == "" {
		return nil, lhttp.NewBadRequest("run_id is required")
	}
	run, err := e.datastore.Remote.GetRun(ctx, *params.ExperimentID, *params.RunID)
	if err != nil {
		return nil, lhttp.NewInternalError(err.Error())
	}
	runParams := make([]*models.ExperimentRunParameter, 0)
	for _, param := range run.Data.Params {
		runParams = append(runParams, &models.ExperimentRunParameter{
			Key:   param.Key,
			Value: param.Value,
		})
	}
	return &runs.GetRunsParametersOK{
		Payload: runParams,
	}, nil
}

func (e ExperimentRunsAPI) PostRuns(ctx context.Context, params runs.PostRunsParams) (*runs.PostRunsOK, *lhttp.HttpError) {
	log.Debugf("deprecated POST handler to register an experiment run invoked.")
	if params.Body == nil {
		return nil, lhttp.NewBadRequest("body is required")
	}
	if params.Body.ExperimentID == "" {
		return nil, lhttp.NewBadRequest("experiment_id is required")
	}
	if params.Body.ExperimentRunID == "" {
		return nil, lhttp.NewBadRequest("experiment_run_id is required")
	}
	run, err := e.db.ExperimentRuns().GetExperimentRun(ctx, params.Body.ExperimentID, params.Body.ExperimentRunID)
	if err != nil {
		return nil, lhttp.NewInternalError(err.Error())
	}
	return &runs.PostRunsOK{
		Payload: &models.ExperimentRun{
			ExperimentID:    params.Body.ExperimentID,
			ExperimentRunID: params.Body.ExperimentRunID,
			ID:              run.Id,
		},
	}, nil
}

func (e ExperimentRunsAPI) DeleteRuns(ctx context.Context, params runs.DeleteRunsParams) (*runs.DeleteRunsOK, *lhttp.HttpError) {
	if params.ExperimentID == nil {
		return nil, lhttp.NewBadRequest("experiment_id is required")
	}
	if params.RunID == nil {
		return nil, lhttp.NewBadRequest("run_id is required")
	}
	err := e.db.ExperimentRuns().DeleteExperimentRun(ctx, *params.ExperimentID, *params.RunID)
	if err != nil {
		return nil, lhttp.NewInternalError(err.Error())
	}
	return &runs.DeleteRunsOK{}, nil
}

func (e ExperimentRunsAPI) Shutdown() error {
	return nil
}
