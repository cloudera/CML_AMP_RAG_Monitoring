package runs

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
	log.Println("beginning runs reconciler resync")

	maxItems := int64(r.config.ResyncMaxItems)

	ids, err := r.db.ExperimentRuns().ListExperimentRunIdsForReconciliation(ctx, maxItems)
	if err != nil {
		log.Printf("failed to fetch run ids: %s", err)
	}
	for _, id := range ids {
		queue.Add(id)
	}

	log.Println(fmt.Sprintf("queueing %d run for reconciliation", len(ids)))

	log.Println("completing reconciler resync")
}

func (r *RunReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	for _, item := range items {
		log.Printf("reconciling run %d", item.ID)
		run, err := r.db.ExperimentRuns().GetExperimentRunById(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch run %d for reconciliation: %s", item.ID, err)
			continue
		}
		// Fetch remote run
		localRun, err := r.dataStores.Local.GetRun(ctx, run.ExperimentId, run.RunId)
		if err != nil {
			log.Printf("failed to fetch run %d from local store: %s", item.ID, err)
			continue
		}
		remoteRun, err := r.dataStores.Remote.GetRun(ctx, run.ExperimentId, run.RemoteRunId)
		if err != nil {
			log.Printf("failed to fetch run %d from remote store: %s", item.ID, err)
			continue
		}
		// Sync the metrics to the remote store
		remoteRun.Data = localRun.Data
		err = r.dataStores.Remote.UpdateRun(ctx, remoteRun)
		if err != nil {
			log.Printf("failed to update run %d in remote store: %s", item.ID, err)
			continue
		}
		// Update the flag and timestamp of the run to indicate that it has completed reconciliation
		err = r.db.ExperimentRuns().UpdateExperimentRunUpdatedAndTimestamp(ctx, run.Id, false, time.Now())
		if err != nil {
			log.Printf("failed to update run %d timestamp: %s", item.ID, err)
			continue
		}
		log.Printf("finished reconciling run %d ", item.ID)
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
