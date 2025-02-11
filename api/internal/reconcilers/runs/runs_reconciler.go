package runs

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
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
		log.Printf("queueing run %d for reconciliation", id)
		queue.Add(id)
	}

	log.Debugln("completing run reconciler resync")
}

func (r *RunReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	log.Printf("reconciling %d experiment runs", len(items))
	for _, item := range items {
		run, err := r.db.ExperimentRuns().GetExperimentRunById(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch run %d for reconciliation: %s", item.ID, err)
			item.Callback(err)
			continue
		}
		log.Printf("reconciling run %s with experiment ID %s and database ID %d", run.ExperimentId, run.RunId, item.ID)

		// Fetch remote run
		remoteRun, err := r.dataStores.Remote.GetRun(ctx, run.ExperimentId, run.RunId)
		if err != nil {
			log.Printf("failed to fetch run %d from remote store: %s", item.ID, err)
			item.Callback(err)
			continue
		}

		// TODO: validate that the run has updated data prior to queueing metrics reconciliation

		// Update the flag and timestamp of the run to indicate that it has completed reconciliation
		err = r.db.ExperimentRuns().MarkExperimentRunForReconciliation(ctx, run.Id, false)
		if err != nil {
			log.Printf("failed to update run %d timestamp: %s", item.ID, err)
			item.Callback(err)
			continue
		}

		// Update the experiment run to indicate that metrics reconciliation is required
		log.Printf("flagging run %s with run ID %s and database ID %d for metrics reconciliation", remoteRun.Info.Name, remoteRun.Info.RunId, item.ID)
		err = r.db.ExperimentRuns().MarkExperimentRunForMetricsReconciliation(ctx, run.Id, true)
		if err != nil {
			log.Printf("failed to update run %d reconcile metrics flag: %s", item.ID, err)
			item.Callback(err)
			continue
		}
		item.Callback(nil)
	}
	log.Printf("finished run reconiliation for %d experiment runs", len(items))
}

func (r *RunReconciler) fetchArtifacts(ctx context.Context, experimentId string, runId string, artifact datasource.Artifact) (map[string][]byte, error) {
	if artifact.IsDir {
		artifacts, err := r.dataStores.Remote.Artifacts(ctx, runId, &artifact.Path)
		if err != nil {
			return nil, err
		}
		result := make(map[string][]byte)
		for _, a := range artifacts {
			children, err := r.fetchArtifacts(ctx, experimentId, runId, a)
			if err != nil {
				return nil, err
			}
			for k, v := range children {
				result[k] = v
			}
		}
		return result, nil
	}
	data, err := r.dataStores.Remote.GetArtifact(ctx, runId, artifact.Path)
	if err != nil {
		return nil, err
	}
	return map[string][]byte{artifact.Path: data}, nil
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
	return "runs-reconciler"
}
