package runs

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/datasource"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/util"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/app"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/reconciler"
	"strings"
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
		log.Debugf("queueing %d runs for reconciliation", len(ids))
	}
	log.Debugln("completing reconciler resync")
}

func (r *RunReconciler) Reconcile(ctx context.Context, items []reconciler.ReconcileItem[int64]) {
	for _, item := range items {
		run, err := r.db.ExperimentRuns().GetExperimentRunById(ctx, item.ID)
		if err != nil {
			log.Printf("failed to fetch run %d for reconciliation: %s", item.ID, err)
			continue
		}
		remoteRunId := run.RemoteRunId
		if remoteRunId == "" {
			remoteRunId = "<undefined>"
		}
		log.Printf("reconciling run %s with experiment ID %s, remote run ID %s, and database ID %d", run.ExperimentId, run.RunId, remoteRunId, item.ID)
		experiment, err := r.db.Experiments().GetExperimentByExperimentId(ctx, run.ExperimentId)
		if err != nil {
			log.Printf("failed to fetch experiment %d for reconciliation: %s", item.ID, err)
			continue
		}
		if experiment.RemoteExperimentId == "" {
			log.Printf("experiment %s with ID %s and database ID %d has no remote experiment id, skipping reconciliation", experiment.Name, experiment.ExperimentId, item.ID)
			continue
		}
		log.Printf("experiment %s with ID %s and database ID %d has remote experiment id %s, syncing run %s", experiment.Name, experiment.ExperimentId, item.ID, experiment.RemoteExperimentId, run.RunId)
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
		log.Printf("syncing data for experiment %s with ID %s run %s to remote store", experiment.Name, experiment.ExperimentId, run.RunId)
		if len(localRun.Data.Metrics) > 0 {
			log.Println("local run metrics: ")
			for _, metric := range localRun.Data.Metrics {
				log.Printf("metric %s: %f, step %d, %s", metric.Key, metric.Value, metric.Step, util.TimeStamp(metric.Timestamp))
			}
		} else {
			log.Printf("local run %s has no metrics", run.RunId)
		}

		// Sync the run to the remote store
		// TODO: verify that data has changed before applying the update
		updated := false
		if remoteRun.Info.Name != localRun.Info.Name {
			remoteRun.Info.Name = localRun.Info.Name
			updated = true
		}
		if remoteRun.Info.Status != localRun.Info.Status {
			remoteRun.Info.Status = localRun.Info.Status
			updated = true
		}
		if remoteRun.Info.EndTime != localRun.Info.EndTime {
			remoteRun.Info.EndTime = localRun.Info.EndTime
			updated = true
		}
		if len(remoteRun.Data.Params) != len(localRun.Data.Params) {
			updated = true
		} else {
			for _, param := range localRun.Data.Params {
				found := false
				for _, remoteParam := range remoteRun.Data.Params {
					if remoteParam.Key == param.Key && remoteParam.Value != param.Value {
						updated = true
					}
					found = true
					break
				}
				if !found {
					updated = true
					break
				}
			}
		}
		if len(remoteRun.Data.Metrics) != len(localRun.Data.Metrics) {
			updated = true
		} else {
			for _, metric := range localRun.Data.Metrics {
				found := false
				for _, remoteMetric := range remoteRun.Data.Metrics {
					if metric.Key == remoteMetric.Key && metric.Step == remoteMetric.Step {
						if metric.Value != remoteMetric.Value || metric.Timestamp != remoteMetric.Timestamp {
							updated = true
						}
						found = true
						break
					}
				}
				if !found {
					updated = true
					break
				}
			}
		}
		if !updated {
			log.Printf("run %s with run ID %s exists in remote store and appears up-to-date", localRun.Info.Name, localRun.Info.RunId)
		} else {
			remoteRun.Data = localRun.Data
			start := time.UnixMilli(localRun.Info.StartTime)
			end := time.UnixMilli(localRun.Info.EndTime)
			remoteStart := time.UnixMilli(remoteRun.Info.StartTime)
			log.Printf("updating run %s in remote store with name %s, status %s, local start time %s (%d), remote start time %s (%d), end time %s (%d), stage %s",
				run.RemoteRunId, remoteRun.Info.Name, string(remoteRun.Info.Status), start, localRun.Info.StartTime, remoteStart, remoteRun.Info.StartTime, end, remoteRun.Info.EndTime, remoteRun.Info.LifecycleStage)
			updatedRun, err := r.dataStores.Remote.UpdateRun(ctx, remoteRun)
			if err != nil {
				log.Printf("failed to update run %d in remote store: %s", item.ID, err)
				continue
			}

			if updatedRun.Info.Name != remoteRun.Info.Name || updatedRun.Info.Status != remoteRun.Info.Status || updatedRun.Info.StartTime != remoteRun.Info.StartTime || updatedRun.Info.EndTime != remoteRun.Info.EndTime || updatedRun.Info.LifecycleStage != remoteRun.Info.LifecycleStage {
				log.Printf("failed to updatedRun run %s info in remote store", run.RemoteRunId)
				if updatedRun.Info.Name != remoteRun.Info.Name {
					log.Printf("name mismatch: %s != %s", updatedRun.Info.Name, remoteRun.Info.Name)
				}
				if updatedRun.Info.Status != remoteRun.Info.Status {
					log.Printf("status mismatch: %s != %s", updatedRun.Info.Status, remoteRun.Info.Status)
				}
				if updatedRun.Info.StartTime != remoteRun.Info.StartTime {
					log.Printf("start time mismatch: %d != %d", updatedRun.Info.StartTime, remoteRun.Info.StartTime)
				}
				if updatedRun.Info.EndTime != remoteRun.Info.EndTime {
					log.Printf("end time mismatch: %d != %d", updatedRun.Info.EndTime, remoteRun.Info.EndTime)
				}
			}
		}

		// sync the metric artifacts
		// first, fetch artifacts from local MLFlow
		log.Printf("fetching artifacts for experiment run %s with database ID %d", run.RunId, item.ID)
		mlFlowArtifacts, err := r.dataStores.Local.Artifacts(ctx, run.RunId, nil)
		if err != nil {
			log.Printf("failed to fetch artifacts for experiment run %s with database ID %d: %s", run.RunId, item.ID, err)
			continue
		}
		for _, artifact := range mlFlowArtifacts {
			log.Printf("syncing artifact %s for experiment run %s with database ID %d", artifact.Path, run.RunId, item.ID)
			if !strings.HasSuffix(artifact.Path, ".json") {
				log.Printf("skipping non-json artifact %s for experiment run %s with database ID %d", artifact.Path, localRun.Info.Name, item.ID)
				continue
			}
			// TODO: filter the json to only sync metric artifacts - need a way to know which artifacts to include/exclude
			metricArtifacts, err := r.fetchArtifacts(ctx, run.ExperimentId, run.RunId, artifact)
			if err != nil {
				log.Printf("failed to fetch artifact %s for experiment run %s with database ID %d: %s", artifact.Path, localRun.Info.Name, item.ID, err)
				continue
			}
			artifactsUpdated := false
			for path, data := range metricArtifacts {
				// sync the artifact to the remote store
				remotePath, uerr := r.dataStores.Remote.UploadArtifact(ctx, experiment.RemoteExperimentId, run.RemoteRunId, path, data)
				if uerr != nil {
					log.Printf("failed to save artifact %s for experiment run %s with database ID %d: %s", path, localRun.Info.Name, item.ID, uerr)
					continue
				}
				log.Printf("saved artifact %s for experiment run %s with database ID %d to remote path %s", path, localRun.Info.Name, item.ID, remotePath)
				found := false
				for _, file := range remoteRun.Data.Files {
					if file.Path == remotePath {
						found = true
						if file.FileSize != int64(len(data)) {
							log.Printf("file size mismatch for artifact %s in remote store: %d != %d", remotePath, file.FileSize, len(data))
							file.FileSize = int64(len(data))
							artifactsUpdated = true
						}
						break
					}
				}
				if !found {
					log.Printf("adding new artifact %s to remote run files", remotePath)
					remoteRun.Data.Files = append(remoteRun.Data.Files, datasource.Artifact{
						Path:     remotePath,
						IsDir:    false,
						FileSize: int64(len(data)),
					})
					artifactsUpdated = true
				}
			}
			if artifactsUpdated {
				log.Printf("updating run %s with %d artifacts", run.RemoteRunId, len(remoteRun.Data.Files))
				updatedRun, uerr := r.dataStores.Remote.UpdateRun(ctx, remoteRun)
				if uerr != nil {
					log.Printf("failed to update run %d with new artifacts: %s", item.ID, uerr)
					continue
				}
				for _, file := range updatedRun.Data.Files {
					log.Printf("updated run %s has artifact %s with size %d", updatedRun.Info.Name, file.Path, file.FileSize)
				}
			} else {
				log.Printf("no new artifacts to update for run %d", item.ID)
			}
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

func (r *RunReconciler) fetchArtifacts(ctx context.Context, experimentId string, runId string, artifact datasource.Artifact) (map[string][]byte, error) {
	if artifact.IsDir {
		artifacts, err := r.dataStores.Local.Artifacts(ctx, runId, &artifact.Path)
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
	data, err := r.dataStores.Local.GetArtifact(ctx, runId, artifact.Path)
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
	return "mlflow-run-reconciler"
}
