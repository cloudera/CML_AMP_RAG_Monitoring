package reconcilers

import (
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/experiments"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/metrics"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/reconcilers/runs"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
)

type ReconcilerSet struct {
	ExperimentReconciler    *experiments.ExperimentReconciler
	ExperimentRunReconciler *experiments.ExperimentRunReconciler
	RunReconciler           *runs.RunReconciler
	MetricsReconciler       *metrics.MetricsReconciler

	experimentManager    *reconciler.Manager[string]
	experimentRunManager *reconciler.Manager[int64]
	runManager           *reconciler.Manager[int64]
	metricsManager       *reconciler.Manager[int64]
}

func NewReconcilerSet(app *app.Instance, experimentsCfg *experiments.Config, experimentReconciler *experiments.ExperimentReconciler, experimentRunReconciler *experiments.ExperimentRunReconciler,
	runCfg *runs.Config, runReconciler *runs.RunReconciler,
	metricsCfg *metrics.Config, metricsReconciler *metrics.MetricsReconciler) *ReconcilerSet {

	experimentManager, err := experiments.NewExperimentReconcilerManager(app, experimentsCfg, experimentReconciler)
	if err != nil {
		panic(err)
	}
	experimentRunManager, err := experiments.NewExperimentRunReconcilerManager(app, experimentsCfg, experimentRunReconciler)
	if err != nil {
		panic(err)
	}
	runManager, err := runs.NewRunReconcilerManager(app, runCfg, runReconciler)
	if err != nil {
		panic(err)
	}
	metricsManager, err := metrics.NewMetricsReconcilerManager(app, metricsCfg, metricsReconciler)
	if err != nil {
		panic(err)
	}

	return &ReconcilerSet{
		ExperimentReconciler:    experimentReconciler,
		ExperimentRunReconciler: experimentRunReconciler,
		RunReconciler:           runReconciler,
		MetricsReconciler:       metricsReconciler,

		experimentManager:    experimentManager,
		experimentRunManager: experimentRunManager,
		runManager:           runManager,
		metricsManager:       metricsManager,
	}
}

func (r *ReconcilerSet) Start() {
	r.experimentManager.Start()
	r.experimentRunManager.Start()
	r.runManager.Start()
	r.metricsManager.Start()
}

func (r *ReconcilerSet) Finish() {
	r.experimentManager.Finish()
	r.experimentRunManager.Finish()
	r.runManager.Finish()
	r.metricsManager.Finish()
}
