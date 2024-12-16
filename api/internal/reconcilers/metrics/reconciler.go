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
	runs, err := r.db.ExperimentRuns().ListExperimentRunIdsForReconciliation(ctx, maxItems)
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
		log.Printf("reconciling experiment run %d", item.ID)
		run, err := r.db.ExperimentRuns().GetExperimentRunById(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch experiment run %d for reconciliation: %s", item.ID, err)
			continue
		}
		// Fetch metrics from MLFlow
		mlFlowMetrics, err := r.mlFlow.Local.Metrics(ctx, run.RunId)
		if err != nil {
			log.Printf("failed to fetch metrics for experiment run %d: %s", item.ID, err)
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
		// Fetch artifacts from MLFlow
		//mlFlowArtifacts, err := r.mlFlow.Local.Artifacts(ctx, run.RunId, nil)
		//if err != nil {
		//	log.Printf("failed to fetch artifacts for experiment run %d: %s", item.ID, err)
		//	continue
		//}
		//for _, artifact := range mlFlowArtifacts {
		//	artifactMetrics, err := r.fetchArtifacts(ctx, run.ExperimentId, run.RunId, artifact)
		//	if err != nil {
		//		log.Printf("failed to fetch artifact %s for experiment run %d: %s", artifact.Path, item.ID, err)
		//		continue
		//	}
		//	log.Printf("fetched %d metrics for artifact %s for experiment run %s", len(artifactMetrics), artifact.Path, run.RunId)
		//}
		// Update the timestamp of the experiment run to indicate that it has been reconciled
		err = r.db.ExperimentRuns().UpdateExperimentRunUpdatedAndTimestamp(ctx, run.Id, true, time.Now())
		if err != nil {
			log.Printf("failed to update experiment run %d timestamp: %s", item.ID, err)
		}
		log.Printf("reconciling metrics for experiment %s and run %s", run.ExperimentId, run.RunId)
	}
}

func (r *Reconciler) fetchArtifacts(ctx context.Context, experimentId string, runId string, artifact datasource.Artifact) ([]db.Metric, error) {
	if artifact.IsDir {
		artifacts, err := r.mlFlow.Local.Artifacts(ctx, runId, &artifact.Path)
		if err != nil {
			return nil, err
		}
		result := make([]db.Metric, 0)
		for _, a := range artifacts {
			metrics, err := r.fetchArtifacts(ctx, experimentId, runId, a)
			if err != nil {
				return nil, err
			}
			result = append(result, metrics...)
		}
	}
	if strings.HasSuffix(artifact.Path, ".json") {
		artifactBytes, err := r.mlFlow.Local.GetArtifact(ctx, runId, artifact.Path)
		if err != nil {
			return nil, err
		}
		value := string(artifactBytes)
		m, err := r.db.Metrics().CreateMetric(ctx, &db.Metric{
			ExperimentId: experimentId,
			RunId:        runId,
			Name:         artifact.Path,
			Type:         db.MetricTypeText,
			ValueText:    &value,
			Tags:         map[string]string{},
		})
		if err != nil {
			log.Printf("failed to insert text metric %s for experiment run %s: %s", artifact.Path, runId, err)
		} else {
			log.Printf("inserted text metric %s(%d) for experiment run %s", m.Name, m.Id, runId)
		}
		return []db.Metric{*m}, nil
	}
	return []db.Metric{}, nil
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
	return "metrics"
}
