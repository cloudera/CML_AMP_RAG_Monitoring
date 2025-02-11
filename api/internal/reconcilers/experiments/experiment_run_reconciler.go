package experiments

import (
	"context"
	"database/sql"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
)

type ExperimentRunReconciler struct {
	config     *Config
	db         db.Database
	dataStores datasource.DataStores
}

func (r *ExperimentRunReconciler) Reboot(_ context.Context) {}

func (r *ExperimentRunReconciler) Resync(ctx context.Context, queue *reconciler.ReconcileQueue[int64]) {
	if !r.config.Enabled {
		return
	}
	log.Println("beginning experiment run reconciler resync")

	maxItems := int64(r.config.ResyncMaxItems)

	ids, err := r.db.Experiments().ListExperimentIDsForReconciliation(ctx, maxItems)
	if err != nil {
		log.Printf("failed to fetch experiments from local mlflow: %s", err)
		return
	}
	for _, id := range ids {
		queue.Add(id)
	}

	log.Printf("queueing %d experiments for run reconciliation", len(ids))

	log.Debugln("completing mlflow sync reconciler resync")
}

func (r *ExperimentRunReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	log.Printf("reconciling %d experiments for experiment runs", len(items))
	for _, item := range items {
		experiment, err := r.db.Experiments().GetExperimentById(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch experiment %d for sync reconciliation: %s", item.ID, err)
			continue
		}

		if experiment == nil || experiment.ExperimentId == "" || experiment.ExperimentId == "0" {
			log.Printf("experiment %d is nil or has no experiment ID, skipping reconciliation", item.ID)
			continue
		}

		log.Printf("reconciling experiment run for experiment %s with experiment ID %s and database ID %d", experiment.Name, experiment.ExperimentId, item.ID)

		remoteRuns, rerr := r.dataStores.Remote.ListRuns(ctx, experiment.ExperimentId)
		if rerr != nil {
			log.Printf("failed to fetch local runs for experiment %s with experiment ID %s and database ID %d: %s", experiment.Name, experiment.ExperimentId, item.ID, rerr)
			continue
		}
		log.Printf("fetched %d remote runs for experiment %s with experiment ID %s and database ID %d", len(remoteRuns), experiment.Name, experiment.ExperimentId, item.ID)
		for _, run := range remoteRuns {
			log.Printf("reconciling run %s with run ID %s", run.Info.Name, run.Info.RunId)

			// Check and see if the run already exists in the DB and insert it if not
			existing, dberr := r.db.ExperimentRuns().GetExperimentRun(ctx, experiment.ExperimentId, run.Info.RunId)
			if dberr != nil && !errors.Is(dberr, sql.ErrNoRows) {
				log.Printf("failed to fetch run %s with run ID %s from DB: %s", run.Info.Name, run.Info.RunId, dberr)
				continue
			}
			var id int64
			if existing != nil {
				log.Printf("run %s with run ID %s found in DB with database ID %d", run.Info.Name, run.Info.RunId, existing.Id)
				id = existing.Id
			} else {
				// Insert the run into the DB
				log.Printf("run %s with run ID %s not found in DB, creating it", run.Info.Name, run.Info.RunId)
				newRun, cdberr := r.db.ExperimentRuns().CreateExperimentRun(ctx, &db.ExperimentRun{
					ExperimentId: run.Info.ExperimentId,
					RunId:        run.Info.RunId,
				})
				if cdberr != nil {
					log.Printf("failed to insert run %s with run ID %s into DB: %s", run.Info.Name, run.Info.RunId, cdberr)
					continue
				}
				id = newRun.Id
			}
			// Flag the run as ready for reconciliation
			log.Printf("flagging run %s with run ID %s and database ID %d for run reconciliation", run.Info.Name, run.Info.RunId, id)
			dberr = r.db.ExperimentRuns().MarkExperimentRunForReconciliation(ctx, id, true)
			if dberr != nil {
				log.Printf("failed to update timestamp for run %s with run ID %s and database ID %d: %s", run.Info.Name, run.Info.RunId, id, dberr)
				continue
			}
		}

		// Update the flag and timestamp of the experiment to indicate that it has finished reconciliation
		udberr := r.db.Experiments().MarkExperimentIDForReconciliation(ctx, experiment.Id, false)
		if udberr != nil {
			log.Printf("failed to clear reconcile flag for experiment %d: %s", item.ID, udberr)
			continue
		}
		log.Printf("finished reconciling experiment runs for experiment %s with experiment ID %s and database ID %d", experiment.Name, experiment.ExperimentId, experiment.Id)
		item.Callback(nil)
	}
	log.Println("finished reconciling experiment run for experiments")
}

func NewExperimentRunReconcilerManager(app *app.Instance, cfg *Config, rec *ExperimentRunReconciler) (*reconciler.Manager[int64], error) {
	log.Println("experiment run reconciler initializing")
	reconcilerConfig, err := reconciler.NewConfig(cfg.ResyncFrequency, cfg.MaxWorkers, cfg.RunMaxItems)

	if err != nil {
		return nil, err
	}
	return reconciler.NewManager[int64](app.Context(), reconcilerConfig, rec), nil
}

func NewExperimentRunReconciler(config *Config, db db.Database, dataStores datasource.DataStores) *ExperimentRunReconciler {
	return &ExperimentRunReconciler{
		config:     config,
		db:         db,
		dataStores: dataStores,
	}
}

func (r *ExperimentRunReconciler) Name() string {
	return "experiment-run-reconciler"
}
