package lsql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

type transactionValueType string

var transactionKey = transactionValueType("transaction")
var transactionValue = true

func setTransaction(ctx context.Context) context.Context {
	return context.WithValue(ctx, transactionKey, transactionValue)
}

func isTransaction(ctx context.Context) bool {
	if v, ok := ctx.Value(transactionKey).(bool); v == transactionValue && ok {
		return true
	}
	return false
}

type Tx struct {
	tx *sqlx.Tx
	db *Instance
}

func (tx *Tx) Transaction(ctx context.Context, callback TransactionFunc) error {
	return callback(ctx, tx)
}

func (tx *Tx) GetDatabaseEngine() string {
	return tx.db.GetDatabaseEngine()
}

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if !isTransaction(ctx) {
		return nil, fmt.Errorf("tried to use transaction without a transaction context")
	}

	if len(args) > 0 {
		var err error
		query, args, err = sqlx.In(query, args...)
		if err != nil {
			return nil, err
		}
	}

	finalQuery := tx.db.db.Rebind(query)

	return tx.tx.ExecContext(ctx, finalQuery, args...)
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *Row {
	if !isTransaction(ctx) {
		return &Row{err: fmt.Errorf("tried to use transaction without a transaction context")}
	}

	if len(args) > 0 {
		var err error
		query, args, err = sqlx.In(query, args...)
		if err != nil {
			return nil
		}
	}

	finalQuery := tx.db.db.Rebind(query)

	return &Row{row: tx.tx.QueryRowxContext(ctx, finalQuery, args...)}
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	if !isTransaction(ctx) {
		return nil, fmt.Errorf("tried to use transaction without a transaction context")
	}

	if len(args) > 0 {
		var err error
		query, args, err = sqlx.In(query, args...)
		if err != nil {
			return nil, err
		}
	}

	finalQuery := tx.db.db.Rebind(query)

	return tx.tx.QueryxContext(ctx, finalQuery, args...)
}

func (tx *Tx) ExecAndReturnId(ctx context.Context, query string, args ...interface{}) (int64, error) {
	return ExecAndReturnId(tx, ctx, query, args...)
}

func (tx *Tx) GenerateLimitCondition(pageSize int64, pageNumber int64, orderByFieldNameForSqlSvr string) (string, error) {
	return tx.db.GenerateLimitCondition(pageSize, pageNumber, orderByFieldNameForSqlSvr)
}

func (tx *Tx) GenerateLimitAndOrderCondition(pageSize int64, pageNumber int64, orderByFieldName string, isDesc bool) (string, error) {
	return tx.db.GenerateLimitAndOrderCondition(pageSize, pageNumber, orderByFieldName, isDesc)
}

type TransactionFunc func(context.Context, *Tx) error

func (db *Instance) Transaction(ctx context.Context, callback TransactionFunc) error {
	if isTransaction(ctx) {
		return fmt.Errorf("can't nest transactions")
	}

	tx, err := db.db.BeginTxx(ctx, nil)
	if err != nil {
		log.Printf("failed to start transaction - %s", err)
		return err
	}

	commited := false
	defer func() {
		if !commited {
			log.Printf("rolling back transaction")
			if err := tx.Rollback(); err != nil {
				log.Printf("failed to rollback transaction - %s", err)
			}
		}
	}()

	if err := callback(setTransaction(ctx), &Tx{tx: tx, db: db}); err != nil {
		//logger.Error("got error during callback", zap.Error(err))
		return err
	}

	if err := tx.Commit(); err != nil {
		log.Printf("failed to commit transaction - %s", err)
		return err
	}
	commited = true

	return nil
}
