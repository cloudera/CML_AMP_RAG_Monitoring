package metrics

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	"strconv"
	"strings"
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

	if len(runs) > 0 {
		log.Debugf("queueing %d experiment runs for metric reconciliation", len(runs))
	}
	for _, run := range runs {
		queue.Add(run)
	}

	log.Debugln("completing reconciler resync")
}

func (r *Reconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	log.Debugf("reconciling %d experiment runs for metrics", len(items))
	for _, item := range items {
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
			log.Printf("failed to fetch experiment run %d for reconciliation: %s", item.ID, err)
			continue
		}
		log.Printf("reconciling metrics for experiment %s with remote ID %s and database ID %d run with remote ID %s and database ID %d",
			experiment.Name, experiment.RemoteExperimentId, item.ID, run.RemoteRunId, run.Id)
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
				log.Printf("inserted numeric metric %s with database ID %d for experiment run %s with database ID %d", m.Name, m.Id, run.RemoteRunId, run.Id)
			}
		}

		// fetch any text metrics stored as json artifacts
		remoteRun, err := r.mlFlow.Remote.GetRun(ctx, experiment.RemoteExperimentId, run.RemoteRunId)
		if err != nil {
			log.Printf("failed to fetch run %s for experiment %s: %s", run.RemoteRunId, experiment.RemoteExperimentId, err)
			continue
		}
		log.Printf("found %d artifacts for experiment run %s", len(remoteRun.Data.Files), run.RemoteRunId)
		for _, artifact := range remoteRun.Data.Files {
			// TODO: filter these
			if strings.HasSuffix(artifact.Path, ".json") {
				log.Printf("fetching artifact %s for experiment run %s", artifact.Path, run.RemoteRunId)
				data, err := r.mlFlow.Remote.GetArtifact(ctx, run.RemoteRunId, artifact.Path)
				if err != nil {
					log.Printf("failed to fetch artifact %s for experiment run %s: %s", artifact.Path, run.RemoteRunId, err)
					continue
				}
				value := string(data)
				textMetric, err := r.db.Metrics().CreateMetric(ctx, &db.Metric{
					ExperimentId: run.ExperimentId,
					RunId:        run.RunId,
					Name:         artifact.Path,
					Type:         db.MetricTypeText,
					ValueText:    &value,
				})
				if err != nil {
					log.Printf("failed to insert text metric %s for experiment run %d: %s", artifact.Path, run.Id, err)
				} else {
					log.Printf("inserted text metric %s with database ID %d for experiment run %s with database ID %d", textMetric.Name, textMetric.Id, run.RemoteRunId, run.Id)
				}
			}
		}

		// Update the metrics flag of the experiment run to indicate that it has been reconciled
		err = r.db.ExperimentRuns().UpdateExperimentRunReconcileMetrics(ctx, run.Id, false)
		if err != nil {
			log.Printf("failed to update experiment run %d for metrics reconciliation: %s", item.ID, err)
		}
		log.Printf("finished reconciling metrics for experiment %s and run %s", run.ExperimentId, run.RunId)
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
