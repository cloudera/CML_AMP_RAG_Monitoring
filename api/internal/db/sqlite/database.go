package sqlite

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.infra.cloudera.com/CAI/AmpRagMonitoring/internal/db"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
)

type Database struct {
	experiments    db.ExperimentService
	experimentRuns db.ExperimentRunService
	metrics        db.MetricsService
}

var _ db.Database = &Database{}

func NewInstance(cfg *lsql.Config) *lsql.Instance {
	if cfg.DatabaseName == "" {
		panic("database name is empty")
	}
	instance, err := lsql.NewInstance(cfg)
	if err != nil {
		log.Printf("failed to create database instance: %s", err)
	}
	if cfg.Engine == "sqlite" {
		_, err = instance.ExecContext(context.Background(), "PRAGMA synchronous=OFF;")
		if err != nil {
			log.Fatal(err)
		}
	}
	return instance
}

func NewDatabase(experiments db.ExperimentService, runs db.ExperimentRunService, metrics db.MetricsService) db.Database {
	return &Database{
		experiments:    experiments,
		experimentRuns: runs,
		metrics:        metrics,
	}
}

func (db *Database) Experiments() db.ExperimentService {
	return db.experiments
}

func (db *Database) ExperimentRuns() db.ExperimentRunService {
	return db.experimentRuns
}

func (db *Database) Metrics() db.MetricsService {
	return db.metrics
}
