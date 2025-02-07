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

// ExperimentReconciler Scan MLFlow for experiments and queue reconciliation for new and updated data
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

	experiments, err := r.dataStores.Remote.ListExperiments(ctx, maxItems, "")
	if err != nil {
		log.Debugf("failed to fetch experiments from local mlflow: %s", err)
	}
	queued := 0
	for _, ex := range experiments {
		if ex.Name == "" || ex.Name == "Default" {
			continue
		}
		// If the experiment is not in the database, add it to the queue
		// TODO: determine which experiments actually need synchronized, currently queuing all of them.
		queued++
		queue.Add(ex.ExperimentId)
	}

	if queued > 0 {
		log.Debugf("queueing %d local experiments for reconciliation", queued)
	}

	log.Debugln("completing reconciler resync")
}

func (r *ExperimentReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[string]) {
	log.Debugf("reconciling %d experiments", len(items))
	for _, item := range items {
		// Fetch the experiment MLFlow
		remote, err := r.dataStores.Remote.GetExperiment(ctx, item.ID)
		if err != nil {
			log.Debugf("failed to fetch experiment %s from mlflow: %s", item.ID, err)
			continue
		}
		log.Debugf("reconciling mlflow experiment %s with experiment ID %s", remote.Name, item.ID)
		// Fetch the experiment from the database
		experiment, err := r.db.Experiments().GetExperimentByExperimentId(ctx, item.ID)
		// If the experiment does not exist in the database, insert it
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("mlflow experiment %s with ID %s not found in database, inserting", remote.Name, item.ID)
				ex, err := r.db.Experiments().CreateExperiment(ctx, remote.ExperimentId, remote.Name, util.TimeStamp(remote.CreatedTime), util.TimeStamp(remote.LastUpdatedTime))
				if err != nil {
					log.Printf("failed to insert experiment %s with ID %s: %s", remote.Name, item.ID, err)
					continue
				}
				log.Printf("finished creating experiment %s with mlflow ID %s.  Database ID is %d", remote.Name, ex.ExperimentId, ex.Id)
				continue
			} else {
				log.Printf("failed to fetch local experiment with %s mlflow ID %s for reconciliation: %s", remote.Name, item.ID, err)
				continue
			}
		}
		log.Printf("flagging experiment %s with ID %s (database ID %d) for run reconciliation", remote.Name, experiment.ExperimentId, experiment.Id)
		err = r.db.Experiments().MarkExperimentIDForReconciliation(ctx, experiment.Id)
		if err != nil {
			log.Printf("failed to update experiment %s with ID %s timestamp: %s", remote.Name, item.ID, err)
		}

		log.Debugf("finished reconciling experiment %s with ID %s and database ID %d", experiment.Name, experiment.ExperimentId, experiment.Id)
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
	return "experiment-reconciler"
}
