package db

import (
	_ "github.com/mattn/go-sqlite3" // Import go-sqlite3 library
)

type Database interface {
	Experiments() ExperimentService
	ExperimentRuns() ExperimentRunService
	Metrics() MetricsService
}
