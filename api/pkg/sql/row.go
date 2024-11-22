package lsql

import (
	"github.com/jmoiron/sqlx"
)

type Row struct {
	err error
	row *sqlx.Row
}

func (r *Row) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	return r.row.Scan(dest...)
}
