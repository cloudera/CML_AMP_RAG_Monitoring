package lsql

import (
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Customizer interface {
	DB(*Config) (*Config, error)
}

type RowScanner interface {
	Scan(...interface{}) error
}

type DBInterface interface {
	GetDatabaseEngine() string
	Transaction(ctx context.Context, callback TransactionFunc) error

	QueryRowContext(ctx context.Context, query string, args ...interface{}) *Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	ExecAndReturnId(ctx context.Context, query string, args ...interface{}) (int64, error)

	// GenerateLimitCondition Use when we don't want sorting for the mysql case. Note that sqlsvr requires a sort to use `FETCH FIRST`.
	GenerateLimitCondition(pageSize int64, pageNumber int64, orderByFieldNameForSqlSvr string) (string, error)
	// GenerateLimitAndOrderCondition Use when we want sorting for both mysql and sqlsvr.
	GenerateLimitAndOrderCondition(pageSize int64, pageNumber int64, orderByFieldName string, isDesc bool) (string, error)
}

var _ DBInterface = &Instance{}
var _ DBInterface = &Tx{}
