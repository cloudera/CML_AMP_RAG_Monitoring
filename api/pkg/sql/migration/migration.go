package lmigration

import (
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	lmigration_sqlite "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql/migration/sqlite"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	"github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/database/sqlserver"
	"github.com/golang-migrate/migrate/v4/source"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	lsql "github.infra.cloudera.com/CAI/AmpRagMonitoring/pkg/sql"
)

type MigrationStatus string

var (
	StatusNotStarted = MigrationStatus("NotStarted")
	StatusRunning    = MigrationStatus("Running")
	StatusDone       = MigrationStatus("Done")
	StatusFailed     = MigrationStatus("Failed")
	StatusCancelled  = MigrationStatus("Cancelled")
)

type Migration struct {
	DB       *sql.DB
	cfg      *lsql.Config
	migrate  *migrate.Migrate
	database database.Driver
	source   source.Driver
	set      MigrationSet
}

type MigrationSet struct {
	AssetNames func() []string
	Asset      func(name string) ([]byte, error)
}

type MigrationLogger struct {
}

func (m MigrationLogger) Printf(format string, v ...interface{}) {
	msg := strings.TrimSpace(fmt.Sprintf(format, v...))
	log.Print(msg)
}

func (m MigrationLogger) Verbose() bool {
	return true
}

func NewMigration(cfg *lsql.Config, sets map[string]MigrationSet) (*Migration, error) {
	set, ok := sets[strings.ToLower(cfg.Engine)]
	if !ok {
		return nil, fmt.Errorf("migration set not found for DB engine: set name: %s", strings.ToLower(cfg.Engine))
	}

	resource := bindata.Resource(set.AssetNames(),
		func(name string) ([]byte, error) {
			return set.Asset(name)
		},
	)

	source, err := bindata.WithInstance(resource)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(cfg.Engine, cfg.PartialAddress())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)

	switch strings.ToLower(cfg.Engine) {
	case "mysql":
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", cfg.DatabaseName))
	case "sqlserver":
		_, err = db.Exec(fmt.Sprintf("IF NOT EXISTS(SELECT 1 FROM sys.databases WHERE name='%s') CREATE DATABASE [%s]",
			cfg.DatabaseName, cfg.DatabaseName))
	case "sqlite":
	default:
		return nil, fmt.Errorf("unsupported DB engine")
	}

	if err != nil {
		return nil, err
	}

	err = db.Close()
	if err != nil {
		return nil, err
	}

	db, err = sql.Open(cfg.Engine, cfg.FullAddress())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(0)

	var database database.Driver
	switch strings.ToLower(cfg.Engine) {
	case "mysql":
		database, err = mysql.WithInstance(db, &mysql.Config{})
	case "sqlserver":
		database, err = sqlserver.WithInstance(db, &sqlserver.Config{})
	case "sqlite":
		database, err = lmigration_sqlite.WithInstance(db, &lmigration_sqlite.Config{})
	default:
		return nil, fmt.Errorf("unknown engine \"%s\"", cfg.Engine)
	}
	if err != nil {
		return nil, err
	}

	mig, err := migrate.NewWithInstance("go-bindata", source, cfg.DatabaseName, database)
	if err != nil {
		return nil, err
	}

	return &Migration{
		DB:       db,
		cfg:      cfg,
		migrate:  mig,
		source:   source,
		set:      set,
		database: database,
	}, nil
}

func (m *Migration) Run(desiredVersion *uint) error {
	// If empty, go to the latest migration. Assumes that migrations come in pairs (up and down), one of which can potentially be empty
	if desiredVersion == nil {
		latestVersion := uint(len(m.set.AssetNames()) / 2)
		desiredVersion = &latestVersion
	}

	version, dirty, err := m.migrate.Version()

	if err != nil && err != migrate.ErrNilVersion {
		return errors.WithStack(err)
	}

	if dirty {
		if version > 1 {
			if err := m.migrate.Force(int(version) - 1); err != nil {
				return errors.WithStack(err)
			}
		} else {
			if err := m.migrate.Drop(); err != nil {
				return errors.WithStack(err)
			}
			m.migrate, err = migrate.NewWithInstance("go-bindata", m.source, m.cfg.DatabaseName, m.database)
			if err != nil {
				return errors.WithStack(err)
			}
		}
	}

	done := make(chan bool)
	errs := make(chan error, 1)

	// Watch for stops
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		signal.Notify(sigint, syscall.SIGTERM)
		select {
		case <-done:
			return
		case <-sigint:
			m.migrate.GracefulStop <- true
		}
	}()

	// Run migration
	go func() {
		if err := m.migrate.Migrate(*desiredVersion); err != nil && err != migrate.ErrNoChange {
			errs <- errors.WithStack(err)
		}
		// Return the main function
		close(errs)

		// Stop watching for interruptions
		close(done)
	}()

	return <-errs
}
