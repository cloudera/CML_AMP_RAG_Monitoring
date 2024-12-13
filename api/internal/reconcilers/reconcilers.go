package reconcilers

import (
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/experiments"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/metrics"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/runs"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
)

type ReconcilerSet struct {
	ExperimentReconciler *experiments.ExperimentReconciler
	SyncReconciler       *experiments.SyncReconciler
	RunReconciler        *runs.RunReconciler
	MetricsReconciler    *metrics.Reconciler

	experimentManager *reconciler.Manager[string]
	syncManager       *reconciler.Manager[int64]
	runManager        *reconciler.Manager[int64]
	metricsManager    *reconciler.Manager[int64]
}

func NewReconcilerSet(app *app.Instance, experimentsCfg *experiments.Config, experimentReconciler *experiments.ExperimentReconciler, syncReconciler *experiments.SyncReconciler,
	runCfg *runs.Config, runReconciler *runs.RunReconciler,
	metricsCfg *metrics.Config, metricsReconciler *metrics.Reconciler) *ReconcilerSet {

	experimentManager, err := experiments.NewExperimentReconcilerManager(app, experimentsCfg, experimentReconciler)
	if err != nil {
		panic(err)
	}
	syncManager, err := experiments.NewSyncReconcilerManager(app, experimentsCfg, syncReconciler)
	if err != nil {
		panic(err)
	}
	runManager, err := runs.NewRunReconcilerManager(app, runCfg, runReconciler)
	if err != nil {
		panic(err)
	}
	metricsManager, err := metrics.NewReconcilerManager(app, metricsCfg, metricsReconciler)
	if err != nil {
		panic(err)
	}

	return &ReconcilerSet{
		ExperimentReconciler: experimentReconciler,
		SyncReconciler:       syncReconciler,
		RunReconciler:        runReconciler,
		MetricsReconciler:    metricsReconciler,

		experimentManager: experimentManager,
		syncManager:       syncManager,
		runManager:        runManager,
		metricsManager:    metricsManager,
	}
}

func (r *ReconcilerSet) Start() {
	r.experimentManager.Start()
	r.syncManager.Start()
	//r.runManager.Start()
	r.metricsManager.Start()
}

func (r *ReconcilerSet) Finish() {
	r.experimentManager.Finish()
	r.syncManager.Finish()
	//r.runManager.Finish()
	r.metricsManager.Finish()
}
