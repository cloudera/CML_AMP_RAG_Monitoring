package metrics

import (
	"context"
	"database/sql"
	"errors"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	"strconv"
	"strings"
	"time"
)

type MetricsReconciler struct {
	config *Config
	db     db.Database
	mlFlow datasource.DataStores
}

func (r *MetricsReconciler) Reboot(_ context.Context) {}

func (r *MetricsReconciler) Resync(ctx context.Context, queue *reconciler.ReconcileQueue[int64]) {
	if !r.config.Enabled {
		return
	}
	log.Debugln("beginning experiment run metrics reconciler resync")

	maxItems := int64(r.config.ResyncMaxItems)
	runs, err := r.db.ExperimentRuns().ListExperimentRunIdsForMetricReconciliation(ctx, maxItems)
	if err != nil {
		log.Errorf("failed to query database: %s", err)
		return
	}

	if len(runs) > 0 {
		log.Printf("queueing %d experiment runs for metric reconciliation", len(runs))
	}
	for _, run := range runs {
		queue.Add(run)
	}

	log.Debugln("completing reconciler resync")
}

func (r *MetricsReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	log.Printf("reconciling %d experiment runs for metrics", len(items))
	for _, item := range items {
		run, dberr := r.db.ExperimentRuns().GetExperimentRunById(ctx, item.ID)
		if dberr != nil {
			log.Printf("failed to fetch experiment run %d for reconciliation: %s", item.ID, dberr)
			item.Callback(dberr)
			continue
		}
		experiment, err := r.db.Experiments().GetExperimentByExperimentId(ctx, run.ExperimentId)
		if err != nil {
			log.Printf("failed to fetch experiment run %d for reconciliation: %s", item.ID, err)
			item.Callback(err)
			continue
		}
		log.Printf("reconciling metrics for experiment %s with ID %s and database ID %d run with ID %s and database ID %d",
			experiment.Name, experiment.ExperimentId, item.ID, run.RunId, run.Id)
		// Fetch metrics from MLFlow
		mlFlowMetrics, err := r.mlFlow.Remote.Metrics(ctx, experiment.ExperimentId, run.RunId)
		if err != nil {
			log.Printf("failed to fetch metrics for experiment run %s: %s", run.RunId, err)
			item.Callback(err)
			continue
		}
		for _, metric := range mlFlowMetrics {
			// platform mlflow metrics use epoch second timestamps
			ts := time.Unix(0, metric.Timestamp*int64(time.Second))
			log.Printf("found metric %s with timestamp %d (%s)", metric.Key, metric.Timestamp, ts)

			// check and see if the metric already exists in the database
			existing, err := r.db.Metrics().GetMetricByName(ctx, run.ExperimentId, run.RunId, metric.Key)
			if err != nil {
				if !errors.Is(err, sql.ErrNoRows) {
					log.Printf("failed to query database: %s", err)
					item.Callback(err)
					continue
				}
			}
			if existing != nil {
				log.Printf("metric %s already exists with database ID %d for experiment run %s", metric.Key, existing.Id, run.RunId)
				if existing.Timestamp == nil || existing.Timestamp.Before(ts) {
					log.Printf("updating timestamp for metric %s with database ID %d for experiment run %s", metric.Key, existing.Id, run.RunId)
					existing.Tags["step"] = strconv.Itoa(metric.Step)
					if existing.Type == db.MetricTypeNumeric {
						existing.ValueNumeric = &metric.Value
					} else {
						log.Printf("metric %s is not numeric, skipping update", metric.Key)
					}
					existing.Timestamp = &ts
					_, err := r.db.Metrics().UpdateMetric(ctx, existing)
					if err != nil {
						log.Printf("failed to update metric %s with database ID %d for experiment run %s: %s", metric.Key, existing.Id, run.RunId, err)
					}
				}
			} else {
				log.Printf("metric %s does not exist in the database for experiment run %s", metric.Key, run.RunId)
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
					item.Callback(err)
				} else {
					log.Printf("inserted numeric metric %s with database ID %d for experiment run %s with database ID %d", m.Name, m.Id, run.RunId, run.Id)
				}
			}
		}

		// fetch any text metrics stored as json artifacts
		remoteRun, err := r.mlFlow.Remote.GetRun(ctx, experiment.ExperimentId, run.RunId)
		if err != nil {
			log.Printf("failed to fetch run %s for experiment %s: %s", run.RunId, experiment.ExperimentId, err)
			item.Callback(err)
			continue
		}
		log.Printf("found %d artifacts for experiment run %s", len(remoteRun.Data.Files), run.RunId)
		for _, artifact := range remoteRun.Data.Files {
			// TODO: filter these
			if strings.HasSuffix(artifact.Path, ".json") {
				log.Printf("fetching artifact %s for experiment run %s", artifact.Path, run.RunId)
				data, err := r.mlFlow.Remote.GetArtifact(ctx, run.RunId, artifact.Path)
				if err != nil {
					log.Printf("failed to fetch artifact %s for experiment run %s: %s", artifact.Path, run.RunId, err)
					item.Callback(err)
					continue
				}
				value := string(data)
				log.Printf("found artifact %s", artifact.Path)
				name := artifact.Path
				lastIndex := strings.LastIndex(name, "/")
				if lastIndex != -1 {
					name = name[lastIndex+1:]
				}
				existing, err := r.db.Metrics().GetMetricByName(ctx, run.ExperimentId, run.RunId, name)
				if err != nil {
					if !errors.Is(err, sql.ErrNoRows) {
						log.Printf("failed to query database: %s", err)
						item.Callback(err)
						continue
					}
				}
				if existing != nil {
					log.Printf("metric %s already exists with database ID %d for experiment run %s", name, existing.Id, run.RunId)
					// Artifacts don't have a timestamp to compare for updates, compare the value instead
					if existing.ValueText != nil && *existing.ValueText != value {
						log.Printf("updating value for metric %s with database ID %d for experiment run %s", name, existing.Id, run.RunId)
						existing.ValueText = &value
						_, err := r.db.Metrics().UpdateMetric(ctx, existing)
						if err != nil {
							log.Printf("failed to update metric %s with database ID %d for experiment run %s: %s", name, existing.Id, run.RunId, err)
							item.Callback(err)
						}
					} else {
						log.Printf("value for metric %s with database ID %d for experiment run %s has not changed", name, existing.Id, run.RunId)
					}
				} else {
					log.Printf("metric %s does not exist in the database for experiment run %s", name, run.RunId)
					textMetric, err := r.db.Metrics().CreateMetric(ctx, &db.Metric{
						ExperimentId: run.ExperimentId,
						RunId:        run.RunId,
						Name:         name,
						Type:         db.MetricTypeText,
						ValueText:    &value,
					})
					if err != nil {
						log.Printf("failed to insert text metric %s for experiment run %d: %s", artifact.Path, run.Id, err)
						item.Callback(err)
						continue
					} else {
						log.Printf("inserted text metric %s with database ID %d for experiment run %s with database ID %d", textMetric.Name, textMetric.Id, run.RunId, run.Id)
					}
				}
			}
		}

		// Update the metrics flag of the experiment run to indicate that it has been reconciled
		err = r.db.ExperimentRuns().MarkExperimentRunForMetricsReconciliation(ctx, run.Id, false)
		if err != nil {
			log.Printf("failed to update experiment run %d for metrics reconciliation: %s", item.ID, err)
			item.Callback(err)
		} else {
			log.Printf("finished reconciling metrics for experiment %s and run %s", run.ExperimentId, run.RunId)
			item.Callback(nil)
		}
	}
}

func NewMetricsReconcilerManager(app *app.Instance, cfg *Config, rec *MetricsReconciler) (*reconciler.Manager[int64], error) {
	log.Println("experiment run metrics reconciler initializing")
	reconcilerConfig, err := reconciler.NewConfig(cfg.ResyncFrequency, cfg.MaxWorkers, cfg.RunMaxItems)

	if err != nil {
		return nil, err
	}
	return reconciler.NewManager[int64](app.Context(), reconcilerConfig, rec), nil
}

func NewMetricsReconciler(config *Config, db db.Database, mlFlow datasource.DataStores) *MetricsReconciler {
	return &MetricsReconciler{
		config: config,
		db:     db,
		mlFlow: mlFlow,
	}
}

func (r *MetricsReconciler) Name() string {
	return "metrics-reconciler"
}
