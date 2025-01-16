package experiments

import (
	"context"
	"database/sql"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/util"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
)

type ExperimentReconciler struct {
	config     *Config
	db         db.Database
	dataStores datasource.DataStores
}

func (r *ExperimentReconciler) Reboot(_ context.Context) {}

func (r *ExperimentReconciler) Resync(ctx context.Context, queue *reconciler.ReconcileQueue[string]) {
	if !r.config.Enabled {
		return
	}
	log.Debugln("beginning experiment reconciler resync")

	maxItems := int64(r.config.ResyncMaxItems)

	// fetch the locally stored experiments to use as a filter
	localExperiments, err := r.db.Experiments().ListExperiments(ctx)
	if err != nil {
		log.Printf("failed to fetch experiments from database: %s", err)
	}

	experiments, err := r.dataStores.Local.ListExperiments(ctx, maxItems, "")
	if err != nil {
		log.Printf("failed to fetch experiments from local mlflow: %s", err)
	}
	queued := 0
	for _, ex := range experiments {
		if ex.Name == "" || ex.Name == "Default" {
			continue
		}
		// If the experiment is not in the database, add it to the queue
		reconcile := true
		found := false
		for _, local := range localExperiments {
			if ex.ExperimentId == local.ExperimentId {
				found = true
				if ex.LastUpdatedTime <= local.UpdatedTs.UnixMilli() {
					reconcile = false
				} else {
					log.Printf("experiment %s with ID %s found in remote mlflow but is out-of-date, queueing for update", ex.Name, ex.ExperimentId)
				}
				break
			}
		}
		if !found {
			log.Printf("experiment %s with mlflow experiment ID %s not found in remote, queueing for creation", ex.Name, ex.ExperimentId)
		}
		if reconcile {
			queued++
			queue.Add(ex.ExperimentId)
		}
	}

	if queued > 0 {
		log.Debugf("queueing %d local experiments for reconciliation", queued)
	}

	log.Debugln("completing reconciler resync")
}

func (r *ExperimentReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[string]) {
	for _, item := range items {
		// Fetch the experiment MLFlow
		local, err := r.dataStores.Local.GetExperiment(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch experiment %s from mlflow: %s", item.ID, err)
			continue
		}
		log.Debugf("reconciling mlflow experiment %s with experiment ID %s,", local.Name, item.ID)
		// Fetch the experiment from the database
		experiment, err := r.db.Experiments().GetExperimentByExperimentId(ctx, item.ID)
		// If the experiment does not exist in the database, insert it
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("mlflow experiment %s with ID %s not found in database, inserting", local.Name, item.ID)
				ex, err := r.db.Experiments().CreateExperiment(ctx, local.ExperimentId, local.Name, util.TimeStamp(local.CreatedTime), util.TimeStamp(local.LastUpdatedTime))
				if err != nil {
					log.Printf("failed to insert experiment %s with ID %s: %s", local.Name, item.ID, err)
					continue
				}
				log.Printf("finished creating experiment %s with mlflow ID %s.  Database ID is %d", local.Name, ex.ExperimentId, ex.Id)
				continue
			} else {
				log.Printf("failed to fetch experiment with %s mlflow ID %s for reconciliation: %s", local.Name, item.ID, err)
				continue
			}
		}
		if experiment == nil {
			log.Printf("mlflow experiment %s with ID %s not found in database, inserting", local.Name, item.ID)
			ex, err := r.db.Experiments().CreateExperiment(ctx, item.ID, local.Name, util.TimeStamp(local.CreatedTime), util.TimeStamp(local.LastUpdatedTime))
			if err != nil {
				log.Printf("failed to insert experiment %s: %s", item.ID, err)
				continue
			}
			log.Printf("finished creating experiment %s with mlflow ID %s.  Database ID is %d", local.Name, ex.ExperimentId, ex.Id)
			continue
		}
		// If the experiment exists in the database, compare the updated timestamps
		lastUpdated := util.TimeStamp(local.LastUpdatedTime)
		updated := false
		if experiment.UpdatedTs.Before(lastUpdated) {
			// Update the flag of the experiment to indicate that it requires reconciliation
			log.Printf("experiment %s with ID %s (database ID %d) is out-of-date, flagging for sync reconciliation", local.Name, experiment.ExperimentId, experiment.Id)
			updated = true
		}
		err = r.db.Experiments().UpdateExperimentUpdatedAndTimestamp(ctx, experiment.Id, updated, lastUpdated)
		if err != nil {
			log.Printf("failed to update experiment %s with ID %s timestamp: %s", local.Name, item.ID, err)
		}

		log.Printf("finished reconciling experiment %s ", experiment.ExperimentId)
	}
}

func NewExperimentReconcilerManager(app *app.Instance, cfg *Config, rec *ExperimentReconciler) (*reconciler.Manager[string], error) {
	log.Println("experiment reconciler initializing")
	reconcilerConfig, err := reconciler.NewConfig(cfg.ResyncFrequency, cfg.MaxWorkers, cfg.RunMaxItems)

	if err != nil {
		return nil, err
	}
	return reconciler.NewManager[string](app.Context(), reconcilerConfig, rec), nil
}

func NewExperimentReconciler(config *Config, db db.Database, dataStores datasource.DataStores) *ExperimentReconciler {
	return &ExperimentReconciler{
		config:     config,
		db:         db,
		dataStores: dataStores,
	}
}

func (r *ExperimentReconciler) Name() string {
	return "mlflow-experiment-reconciler"
}
