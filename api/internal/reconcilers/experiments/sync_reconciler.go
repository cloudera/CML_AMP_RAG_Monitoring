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
	log.Println("beginning experiment sync reconciler resync")

	maxItems := int64(r.config.ResyncMaxItems)

	ids, err := r.db.Experiments().ListExperimentIDsForReconciliation(ctx, maxItems)
	if err != nil {
		log.Printf("failed to fetch experiments from local mlflow: %s", err)
		return
	}
	for _, id := range ids {
		queue.Add(id)
	}

	if len(ids) > 0 {
		log.Printf("queueing %d experiments for sync reconciliation", len(ids))
	}
	log.Debugln("completing mlflow sync reconciler resync")
}

func (r *SyncReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	log.Printf("sync reconciling %d experiments", len(items))
	for _, item := range items {
		experiment, err := r.db.Experiments().GetExperimentById(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch experiment %d for sync reconciliation: %s", item.ID, err)
		}

		if experiment == nil || experiment.ExperimentId == "" || experiment.ExperimentId == "0" {
			continue
		}

		log.Printf("sync reconciling experiment %s with experiment ID %s and database ID %d", experiment.Name, experiment.ExperimentId, item.ID)
		local, err := r.dataStores.Local.GetExperiment(ctx, experiment.ExperimentId)
		if err != nil {
			log.Printf("failed to fetch experiment %d from local store for sync reconciliation: %s", item.ID, err)
			continue
		}

		if experiment.RemoteExperimentId == "" {
			// If the experiment does not exist in the remote store, insert it
			log.Printf("experiment %s with experiment ID %s and database ID %d has no remote experiment ID, inserting into the remote MLFLow instance", experiment.Name, experiment.ExperimentId, item.ID)
			remoteExperimentId, err := r.dataStores.Remote.CreateExperiment(ctx, local.Name)
			if err != nil {
				log.Printf("failed to insert experiment %s with experiment ID %s and database ID %d into remote store: %s", experiment.Name, experiment.ExperimentId, item.ID, err)
				continue
			}
			err = r.db.Experiments().UpdateRemoteExperimentId(ctx, experiment.Id, remoteExperimentId)
			if err != nil {
				log.Printf("failed to update experiment %s with experiment ID %s and database ID %d remote experiment ID: %s", experiment.Name, experiment.ExperimentId, item.ID, err)
				continue
			}
			ex, err := r.db.Experiments().GetExperimentById(ctx, item.ID)
			if err != nil {
				log.Printf("failed to fetch experiment %s with experiment ID %s and database ID %d for reconciliation: %s", experiment.Name, experiment.ExperimentId, item.ID, err)
			}
			experiment = ex
		}
		remoteExperiment, err := r.dataStores.Remote.GetExperiment(ctx, experiment.RemoteExperimentId)
		if err != nil {
			log.Errorf("failed to fetch experiment %s with experiment ID %s and database ID %d from remote store: %s", experiment.Name, experiment.ExperimentId, item.ID, err)
			continue
		}

		// sync the experiment from the local store to the remote store
		if remoteExperiment == nil {
			log.Printf("experiment %s with experiment ID %s and database ID %d not found in remote store, inserting", experiment.Name, experiment.ExperimentId, item.ID)
			remoteExperimentId, err := r.dataStores.Remote.CreateExperiment(ctx, local.Name)
			if err != nil {
				log.Printf("failed to insert experiment %s with experiment ID %s and database ID %d into remote store: %s", experiment.Name, experiment.ExperimentId, item.ID, err)
				continue
			}
			err = r.db.Experiments().UpdateRemoteExperimentId(ctx, experiment.Id, remoteExperimentId)
		}

		// Fetch the experiment runs from local MLFlow
		localRuns, err := r.dataStores.Local.ListRuns(ctx, experiment.ExperimentId)
		if err != nil {
			log.Printf("failed to fetch local runs for experiment %s with experiment ID %s and database ID %d: %s", experiment.Name, experiment.ExperimentId, item.ID, err)
			continue
		}
		remoteRuns, err := r.dataStores.Remote.ListRuns(ctx, experiment.RemoteExperimentId)
		if err != nil {
			log.Printf("failed to fetch local runs for experiment %s with experiment ID %s and database ID %d: %s", experiment.Name, experiment.ExperimentId, item.ID, err)
			continue
		}
		for _, run := range localRuns {
			log.Printf("reconciling local run %s with run ID %s", run.Info.Name, run.Info.RunId)
			found := false
			updated := false
			for _, remoteRun := range remoteRuns {
				if run.Info.Name == remoteRun.Info.Name {
					log.Printf("run %s with run ID %s exists in remote store with run ID %s", run.Info.Name, run.Info.RunId, remoteRun.Info.RunId)
					found = true
					if run.Info.StartTime > remoteRun.Info.StartTime {
						log.Printf("run %s with run ID %s exists in remote store with ID %s but is out of date", run.Info.Name, run.Info.RunId, remoteRun.Info.RunId)
						updated = true
					}
					break
				}
			}
			if found {
				if !updated {
					log.Printf("run %s with run ID %s exists in remote store and appears up-to-date", run.Info.Name, run.Info.RunId)
					//continue
				} else {
					log.Printf("run %s with run ID %s exists in remote store but is out of date", run.Info.Name, run.Info.RunId)
				}
			}
			var remoteRunId string
			if !found {
				// Insert the run into the remote store
				log.Printf("run %s with run ID %s not found in remote store, creating it", run.Info.Name, run.Info.RunId)
				id, err := r.dataStores.Remote.CreateRun(ctx, experiment.RemoteExperimentId, run.Info.Name, util.TimeStamp(run.Info.StartTime), run.Data.Tags)
				if err != nil {
					log.Printf("failed to insert run %s with run ID %s into remote store: %s", run.Info.Name, run.Info.RunId, err)
					continue
				}
				remoteRunId = id
			} else {
				remoteRunId = run.Info.RunId
			}
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
				newRun, dberr := r.db.ExperimentRuns().CreateExperimentRun(ctx, &db.ExperimentRun{
					Id:           0,
					ExperimentId: run.Info.ExperimentId,
					RunId:        run.Info.RunId,
					RemoteRunId:  remoteRunId,
				})
				if dberr != nil {
					log.Printf("failed to insert run %s with run ID %s into DB: %s", run.Info.Name, run.Info.RunId, dberr)
					continue
				}
				id = newRun.Id
			}
			// Flag the run as ready for reconciliation
			log.Printf("flagging run %s with run ID %s and database ID %d for run reconciliation", run.Info.Name, run.Info.RunId, id)
			dberr = r.db.ExperimentRuns().UpdateExperimentRunUpdatedAndTimestamp(ctx, id, true, time.Now())
			if dberr != nil {
				log.Printf("failed to update timestamp for run %s with run ID %s and database ID %d: %s", run.Info.Name, run.Info.RunId, id, dberr)
			}
		}

		// Update the flag and timestamp of the experiment to indicate that it has finished reconciliation
		err = r.db.Experiments().UpdateExperimentUpdatedAndTimestamp(ctx, experiment.Id, false, util.TimeStamp(local.LastUpdatedTime))
		if err != nil {
			log.Printf("failed to update experiment %d timestamp: %s", item.ID, err)
		}
		log.Printf("finished sync reconciling experiment %s with experiment ID %s and database ID %d", experiment.Name, experiment.ExperimentId, experiment.Id)
	}
}

func NewSyncReconcilerManager(app *app.Instance, cfg *Config, rec *SyncReconciler) (*reconciler.Manager[int64], error) {
	log.Println("experiment sync reconciler initializing")
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
