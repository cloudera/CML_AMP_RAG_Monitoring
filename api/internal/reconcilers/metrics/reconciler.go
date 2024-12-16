package metrics

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	"strconv"
	"time"
)

type Reconciler struct {
	config *Config
	db     db.Database
	mlFlow datasource.DataStores
}

func (r *Reconciler) Reboot(_ context.Context) {}

func (r *Reconciler) Resync(ctx context.Context, queue *reconciler.ReconcileQueue[int64]) {
	if !r.config.Enabled {
		return
	}
	log.Debugln("beginning experiment run metrics reconciler resync")

	maxItems := int64(r.config.ResyncMaxItems)
	runs, err := r.db.ExperimentRuns().ListExperimentRunIdsForMetricReconciliation(ctx, maxItems)
	if err != nil {
		log.Printf("failed to query database: %s", err)
		return
	}

	log.Printf("queueing %d experiment runs for metric reconciliation", len(runs))

	for _, run := range runs {
		queue.Add(run)
	}

	log.Debugln("completing reconciler resync")
}

func (r *Reconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	for _, item := range items {
		log.Debugf("reconciling metrics for experiment run %d", item.ID)
		run, dberr := r.db.ExperimentRuns().GetExperimentRunById(ctx, item.ID)
		if dberr != nil {
			log.Printf("failed to fetch experiment run %d for reconciliation: %s", item.ID, dberr)
			continue
		}
		if run.RemoteRunId == "" {
			log.Printf("experiment run %d has no remote run id, skipping reconciliation", item.ID)
			continue
		}
		experiment, err := r.db.Experiments().GetExperimentByExperimentId(ctx, run.ExperimentId)
		if err != nil {
			log.Printf("failed to fetch experiment %d for reconciliation: %s", item.ID, err)
			continue
		}
		log.Printf("reconciling metrics for experiment %s run (%d) %s", experiment.RemoteExperimentId, item.ID, run.RemoteRunId)
		// Fetch metrics from MLFlow
		mlFlowMetrics, err := r.mlFlow.Remote.Metrics(ctx, experiment.RemoteExperimentId, run.RemoteRunId)
		if err != nil {
			log.Printf("failed to fetch metrics for experiment run %s: %s", run.RemoteRunId, err)
			continue
		}
		for _, metric := range mlFlowMetrics {
			ts := time.Unix(0, metric.Timestamp*int64(time.Millisecond))
			m, err := r.db.Metrics().CreateMetric(ctx, &db.Metric{
				ExperimentId: run.ExperimentId,
				RunId:        run.RunId,
				Name:         metric.Key,
				Type:         db.MetricTypeNumeric,
				ValueNumeric: &metric.Value,
				Tags: map[string]string{
					"step": strconv.Itoa(metric.Step),
				},
				Timestamp: &ts,
			})
			if err != nil {
				log.Printf("failed to insert numeric metric %s for experiment run %d: %s", metric.Key, run.Id, err)
			} else {
				log.Printf("inserted numeric metric %s(%d) for experiment run %d", m.Name, m.Id, run.Id)
			}
		}
		// Update the metrics flag of the experiment run to indicate that it has been reconciled
		err = r.db.ExperimentRuns().UpdateExperimentRunReconcileMetrics(ctx, run.Id, false)
		if err != nil {
			log.Printf("failed to update experiment run %d for metrics reconciliation: %s", item.ID, err)
		}
		log.Debugf("reconciling metrics for experiment %s and run %s", run.ExperimentId, run.RunId)
	}
}

func NewReconcilerManager(app *app.Instance, cfg *Config, rec *Reconciler) (*reconciler.Manager[int64], error) {
	log.Println("experiment run metrics reconciler initializing")
	reconcilerConfig, err := reconciler.NewConfig(cfg.ResyncFrequency, cfg.MaxWorkers, cfg.RunMaxItems)

	if err != nil {
		return nil, err
	}
	return reconciler.NewManager[int64](app.Context(), reconcilerConfig, rec), nil
}

func NewReconciler(config *Config, db db.Database, mlFlow datasource.DataStores) *Reconciler {
	return &Reconciler{
		config: config,
		db:     db,
		mlFlow: mlFlow,
	}
}

func (r *Reconciler) Name() string {
	return "metrics-reconciler"
}
