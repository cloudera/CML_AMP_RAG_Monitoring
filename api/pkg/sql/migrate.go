package lsql

import (
	"github.com/golang-migrate/migrate/v4"
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
)

type MigrationSet struct {
	AssetNames func() []string
	Asset      func(name string) ([]byte, error)
}

func Migrate(desiredVersion uint, migrations MigrationSet, address string) error {
	s := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		},
	)

	d, err := bindata.WithInstance(s)
	if err != nil {
		return err
	}
	m, err := migrate.NewWithSourceInstance("go-bindata", d, address)
	if err != nil {
		return err
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return err
	}

	if dirty {
		if version > 1 {
			if err := m.Force(int(version) - 1); err != nil {
				return err
			}
		} else {
			if err := m.Drop(); err != nil {
				return err
			}
			m, err = migrate.NewWithSourceInstance("go-bindata", d, address)
			if err != nil {
				return err
			}
		}
	}

	if err := m.Migrate(desiredVersion); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}
