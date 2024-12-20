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
	for _, ex := range experiments {
		if ex.Name == "" || ex.Name == "Default" {
			continue
		}
		// If the experiment is not in the database, add it to the queue
		reconcile := true
		for _, local := range localExperiments {
			if ex.ExperimentId == local.ExperimentId {
				if ex.LastUpdatedTime <= local.UpdatedTs.UnixMilli() {
					reconcile = false
				}
				break
			}
		}
		if reconcile {
			queue.Add(ex.ExperimentId)
		}
	}

	log.Printf("queueing %d local experiments for reconciliation", len(experiments))

	log.Debugln("completing reconciler resync")
}

func (r *ExperimentReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[string]) {
	for _, item := range items {
		log.Debugf("reconciling experiment %s", item.ID)
		// Fetch the experiment MLFlow
		local, err := r.dataStores.Local.GetExperiment(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch experiment %s from mlflow: %s", item.ID, err)
			continue
		}
		// Fetch the experiment from the database
		experiment, err := r.db.Experiments().GetExperimentByExperimentId(ctx, item.ID)
		// If the experiment does not exist in the database, insert it
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("experiment %s not found in database, inserting", item.ID)
				ex, err := r.db.Experiments().CreateExperiment(ctx, local.ExperimentId, util.TimeStamp(local.CreatedTime), util.TimeStamp(local.LastUpdatedTime))
				if err != nil {
					log.Printf("failed to insert experiment %s: %s", item.ID, err)
					continue
				}
				log.Printf("finished creating experiment %s ", ex.ExperimentId)
				continue
			} else {
				log.Printf("failed to fetch experiment %s for reconciliation: %s", item.ID, err)
				continue
			}
		}
		if experiment == nil {
			log.Printf("experiment %s not found in database, inserting", item.ID)
			ex, err := r.db.Experiments().CreateExperiment(ctx, item.ID, util.TimeStamp(local.CreatedTime), util.TimeStamp(local.LastUpdatedTime))
			if err != nil {
				log.Printf("failed to insert experiment %s: %s", item.ID, err)
				continue
			}
			log.Printf("finished creating experiment %s ", ex.ExperimentId)
			continue
		}
		// If the experiment exists in the database, compare the updated timestamps
		lastUpdated := util.TimeStamp(local.LastUpdatedTime)
		updated := false
		if experiment.UpdatedTs.Before(lastUpdated) {
			// Update the flag of the experiment to indicate that it requires reconciliation
			updated = true
		}
		err = r.db.Experiments().UpdateExperimentUpdatedAndTimestamp(ctx, experiment.Id, updated, lastUpdated)
		if err != nil {
			log.Printf("failed to update experiment %s timestamp: %s", item.ID, err)
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
