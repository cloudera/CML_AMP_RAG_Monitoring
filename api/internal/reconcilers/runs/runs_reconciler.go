package runs

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/util"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	"time"
)

type RunReconciler struct {
	config     *Config
	db         db.Database
	dataStores datasource.DataStores
}

func (r *RunReconciler) Reboot(_ context.Context) {}

func (r *RunReconciler) Resync(ctx context.Context, queue *reconciler.ReconcileQueue[int64]) {
	if !r.config.Enabled {
		return
	}
	log.Debugln("beginning runs reconciler resync")

	maxItems := int64(r.config.ResyncMaxItems)

	ids, err := r.db.ExperimentRuns().ListExperimentRunIdsForReconciliation(ctx, maxItems)
	if err != nil {
		log.Printf("failed to fetch run ids: %s", err)
	}
	for _, id := range ids {
		queue.Add(id)
	}

	if len(ids) > 0 {
		log.Debugln(fmt.Sprintf("queueing %d runs for reconciliation", len(ids)))
	}
	log.Debugln("completing reconciler resync")
}

func (r *RunReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	for _, item := range items {
		log.Printf("reconciling run %d", item.ID)
		run, err := r.db.ExperimentRuns().GetExperimentRunById(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch run %d for reconciliation: %s", item.ID, err)
			continue
		}
		experiment, err := r.db.Experiments().GetExperimentByExperimentId(ctx, run.ExperimentId)
		if err != nil {
			log.Printf("failed to fetch experiment %d for reconciliation: %s", item.ID, err)
			continue
		}
		if experiment.RemoteExperimentId == "" {
			log.Printf("experiment %s(%d) has no remote experiment id, skipping reconciliation", experiment.ExperimentId, item.ID)
			continue
		}
		log.Printf("experiment %s(%d) has remote experiment id %s, syncing run %s", experiment.ExperimentId, item.ID, experiment.RemoteExperimentId, run.RunId)
		// Fetch remote run
		localRun, err := r.dataStores.Local.GetRun(ctx, run.ExperimentId, run.RunId)
		if err != nil {
			log.Printf("failed to fetch run %d from local store: %s", item.ID, err)
			continue
		}
		var remoteRun *datasource.Run
		if run.RemoteRunId == "" {
			// create the remote run
			runId, err := r.dataStores.Remote.CreateRun(ctx, experiment.RemoteExperimentId, localRun.Info.Name, util.TimeStamp(localRun.Info.StartTime), localRun.Data.Tags)
			if err != nil {
				log.Printf("failed to create run %d in remote store: %s", item.ID, err)
				continue
			}
			newRun, err := r.dataStores.Remote.GetRun(ctx, experiment.RemoteExperimentId, runId)
			if err != nil {
				log.Printf("failed to fetch run %d from remote store: %s", item.ID, err)
				continue
			}
			dberr := r.db.ExperimentRuns().UpdateRemoteRunId(ctx, run.Id, runId)
			if dberr != nil {
				log.Printf("failed to update run %d remote run ID: %s", item.ID, dberr)
				continue
			}
			run.RemoteRunId = runId
			remoteRun = newRun
		} else {
			existing, err := r.dataStores.Remote.GetRun(ctx, experiment.RemoteExperimentId, run.RemoteRunId)
			if err != nil {
				log.Printf("failed to fetch run %d from remote store: %s", item.ID, err)
				continue
			}
			remoteRun = existing
		}
		log.Printf("syncing data for run %s to remote store", run.RunId)
		// Sync the metrics to the remote store
		remoteRun.Info.Name = localRun.Info.Name
		remoteRun.Info.Status = localRun.Info.Status
		remoteRun.Info.StartTime = util.TimeStamp(localRun.Info.StartTime).UnixMilli()
		remoteRun.Info.EndTime = util.TimeStamp(localRun.Info.EndTime).UnixMilli()
		remoteRun.Info.LifecycleStage = localRun.Info.LifecycleStage
		remoteRun.Data = localRun.Data
		log.Printf("updating run %s in remote store with name %s, status %s, start time %d, end time %d, stage %s", run.RemoteRunId, remoteRun.Info.Name, string(remoteRun.Info.Status), remoteRun.Info.StartTime, remoteRun.Info.EndTime, remoteRun.Info.LifecycleStage)
		err = r.dataStores.Remote.UpdateRun(ctx, remoteRun)
		if err != nil {
			log.Printf("failed to update run %d in remote store: %s", item.ID, err)
			continue
		}

		// fetch back the run to verify the updates
		verify, verr := r.dataStores.Remote.GetRun(ctx, experiment.RemoteExperimentId, run.RemoteRunId)
		if verr != nil {
			log.Printf("failed to fetch run %s from remote store: %s", run.RemoteRunId, verr)
			continue
		}
		if verify.Info.Name != remoteRun.Info.Name || verify.Info.Status != remoteRun.Info.Status || verify.Info.StartTime != remoteRun.Info.StartTime || verify.Info.EndTime != remoteRun.Info.EndTime || verify.Info.LifecycleStage != remoteRun.Info.LifecycleStage {
			log.Printf("failed to verify run %s info in remote store", run.RemoteRunId)
			if verify.Info.Name != remoteRun.Info.Name {
				log.Printf("name mismatch: %s != %s", verify.Info.Name, remoteRun.Info.Name)
			}
			if verify.Info.Status != remoteRun.Info.Status {
				log.Printf("status mismatch: %s != %s", verify.Info.Status, remoteRun.Info.Status)
			}
			if verify.Info.StartTime != remoteRun.Info.StartTime {
				log.Printf("start time mismatch: %d != %d", verify.Info.StartTime, remoteRun.Info.StartTime)
			}
			if verify.Info.EndTime != remoteRun.Info.EndTime {
				log.Printf("end time mismatch: %d != %d", verify.Info.EndTime, remoteRun.Info.EndTime)
			}
			if verify.Info.LifecycleStage != remoteRun.Info.LifecycleStage {
				log.Printf("lifecycle stage mismatch: %s != %s", verify.Info.LifecycleStage, remoteRun.Info.LifecycleStage)
			}
			continue
		}
		if len(verify.Data.Metrics) != len(remoteRun.Data.Metrics) {
			log.Printf("failed to verify run %s data in remote store", run.RemoteRunId)
			continue
		}
		// Update the flag and timestamp of the run to indicate that it has completed reconciliation
		err = r.db.ExperimentRuns().UpdateExperimentRunUpdatedAndTimestamp(ctx, run.Id, false, time.Now())
		if err != nil {
			log.Printf("failed to update run %d timestamp: %s", item.ID, err)
			continue
		}

		// Update the experiment run to indicate that metrics reconciliation is required
		err = r.db.ExperimentRuns().UpdateExperimentRunReconcileMetrics(ctx, run.Id, true)
		if err != nil {
			log.Printf("failed to update run %d reconcile metrics flag: %s", item.ID, err)
			continue
		}
		log.Debugf("finished reconciling run %d ", item.ID)
	}
}

func NewRunReconcilerManager(app *app.Instance, cfg *Config, rec *RunReconciler) (*reconciler.Manager[int64], error) {
	log.Println("mlflow run reconciler initializing")
	reconcilerConfig, err := reconciler.NewConfig(cfg.ResyncFrequency, cfg.MaxWorkers, cfg.RunMaxItems)

	if err != nil {
		return nil, err
	}
	return reconciler.NewManager[int64](app.Context(), reconcilerConfig, rec), nil
}

func NewRunReconciler(config *Config, db db.Database, dataStores datasource.DataStores) *RunReconciler {
	return &RunReconciler{
		config:     config,
		db:         db,
		dataStores: dataStores,
	}
}

func (r *RunReconciler) Name() string {
	return "mlflow-run-reconciler"
}
