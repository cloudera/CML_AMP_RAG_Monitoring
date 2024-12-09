package experiments

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	"time"
)

type SyncReconciler struct {
	config     *Config
	db         db.Database
	dataStores datasource.DataStores
}

func (r *SyncReconciler) Reboot(_ context.Context) {}

func (r *SyncReconciler) Resync(ctx context.Context, queue *reconciler.ReconcileQueue[int64]) {
	if !r.config.Enabled {
		return
	}
	log.Println("beginning experiments reconciler resync")

	maxItems := int64(r.config.ResyncMaxItems)

	ids, err := r.db.Experiments().ListExperimentIDsForReconciliation(ctx, maxItems)
	if err != nil {
		log.Printf("failed to fetch experiments from local mlflow: %s", err)
	}
	for _, id := range ids {
		queue.Add(id)
	}

	log.Println(fmt.Sprintf("queueing %d experiments for sync reconciliation", len(ids)))

	log.Println("completing mlflow sync reconciler resync")
}

func (r *SyncReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	for _, item := range items {
		log.Printf("reconciling experiment %d", item.ID)
		experiment, err := r.db.Experiments().GetExperimentById(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch experiment %d for reconciliation: %s", item.ID, err)
		}

		local, err := r.dataStores.Local.GetExperiment(ctx, experiment.ExperimentId)
		if err != nil {
			log.Printf("failed to fetch experiment %d from local store: %s", item.ID, err)
			continue
		}

		if experiment.RemoteExperimentId == "" {
			// If the experiment does not exist in the remote store, insert it
			log.Printf("experiment %d not found in remote store, inserting", item.ID)
			remoteExperimentId, err := r.dataStores.Remote.CreateExperiment(ctx, local.Name)
			if err != nil {
				log.Printf("failed to insert experiment %d into remote store: %s", item.ID, err)
				continue
			}
			err = r.db.Experiments().UpdateRemoteExperimentId(ctx, experiment.Id, remoteExperimentId)
			if err != nil {
				log.Printf("failed to update experiment %d remote experiment ID: %s", item.ID, err)
				continue
			}
			ex, err := r.db.Experiments().GetExperimentById(ctx, item.ID)
			if err != nil {
				log.Printf("failed to fetch experiment %d for reconciliation: %s", item.ID, err)
			}
			experiment = ex
		}
		remoteExperiment, err := r.dataStores.Remote.GetExperiment(ctx, experiment.RemoteExperimentId)
		if err != nil {
			log.Errorf("failed to fetch experiment %d from remote store: %s", item.ID, err)
			continue
		}

		// sync the experiment from the local store to the remote store
		if remoteExperiment == nil {
			log.Printf("experiment %d not found in remote store, inserting", item.ID)
			remoteExperimentId, err := r.dataStores.Remote.CreateExperiment(ctx, local.Name)
			if err != nil {
				log.Printf("failed to insert experiment %d into remote store: %s", item.ID, err)
				continue
			}
			err = r.db.Experiments().UpdateRemoteExperimentId(ctx, experiment.Id, remoteExperimentId)
		}

		// Fetch the experiment runs from local MLFlow
		localRuns, err := r.dataStores.Local.ListRuns(ctx, experiment.ExperimentId)
		if err != nil {
			log.Printf("failed to fetch local runs for experiment %d: %s", item.ID, err)
			continue
		}
		remoteRuns, err := r.dataStores.Remote.ListRuns(ctx, experiment.RemoteExperimentId)
		if err != nil {
			log.Printf("failed to fetch local runs for experiment %d: %s", item.ID, err)
			continue
		}
		for _, run := range localRuns {
			found := false
			for _, remoteRun := range remoteRuns {
				if run.Info.Name == remoteRun.Info.Name {
					found = true
					break
				}
			}
			if found {
				continue
			}
			// Insert the run into the remote store
			remoteRunId, err := r.dataStores.Remote.CreateRun(ctx, experiment.RemoteExperimentId, run.Info.Name, ts(run.Info.StartTime), run.Data.Tags)
			if err != nil {
				log.Printf("failed to insert run %s into remote store: %s", run.Info.Name, err)
				continue
			}
			// Insert the run into the DB
			newRun, err := r.db.ExperimentRuns().CreateExperimentRun(ctx, &db.ExperimentRun{
				Id:           0,
				ExperimentId: run.Info.ExperimentId,
				RunId:        run.Info.RunId,
				RemoteRunId:  remoteRunId,
			})
			if err != nil {
				log.Printf("failed to insert run %s into DB: %s", run.Info.Name, err)
				continue
			}
			// Flag the run as ready for reconciliation
			err = r.db.ExperimentRuns().UpdateExperimentRunUpdatedAndTimestamp(ctx, newRun.Id, true, time.Now())
			if err != nil {
				log.Printf("failed to update run %d timestamp: %s", newRun.Id, err)
			}
		}

		// Update the flag and timestamp of the experiment to indicate that it has finished reconciliation
		err = r.db.Experiments().UpdateExperimentUpdatedAndTimestamp(ctx, experiment.Id, false, ts(local.LastUpdatedTime))
		if err != nil {
			log.Printf("failed to update experiment %d timestamp: %s", item.ID, err)
		}
		log.Printf("finished reconciling experiment %s ", experiment.ExperimentId)
	}
}

func NewSyncReconcilerManager(app *app.Instance, cfg *Config, rec *SyncReconciler) (*reconciler.Manager[int64], error) {
	log.Println("experiment run metrics reconciler initializing")
	reconcilerConfig, err := reconciler.NewConfig(cfg.ResyncFrequency, cfg.MaxWorkers, cfg.RunMaxItems)

	if err != nil {
		return nil, err
	}
	return reconciler.NewManager[int64](app.Context(), reconcilerConfig, rec), nil
}

func NewSyncReconciler(config *Config, db db.Database, dataStores datasource.DataStores) *SyncReconciler {
	return &SyncReconciler{
		config:     config,
		db:         db,
		dataStores: dataStores,
	}
}

func (r *SyncReconciler) Name() string {
	return "mlflow-sync-reconciler"
}
