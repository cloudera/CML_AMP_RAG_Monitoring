package lsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"strings"
)

const sqlServerKeyConstraintErrorCode = 2627
const netPeerAddressKey = attribute.Key("net.peer.address")

var (
	ErrConstraintViolation = errors.New("constraint violation")
)

func NewInstance(cfg *Config) (*Instance, error) {
	db, err := sqlx.Connect(cfg.Engine, cfg.FullAddress())
	if err != nil {
		return nil, err
	}

	db.SetConnMaxLifetime(cfg.MaxLifetime)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetMaxOpenConns(cfg.MaxOpenConns)

	tracer := otel.Tracer("lsql")

	return &Instance{
		cfg:    cfg,
		db:     db,
		tracer: tracer,
	}, nil
}

type Instance struct {
	cfg               *Config
	db                *sqlx.DB
	filePathFormatter func(string) string
	tracer            trace.Tracer
}

func (db *Instance) GetDatabaseEngine() string {
	return db.cfg.Engine
}

func (db *Instance) Ping(ctx context.Context) error {
	return db.db.PingContext(ctx)
}

func (db *Instance) Close() error {
	return db.db.Close()
}

func startSpan(ctx context.Context, db *Instance, spanName string, query string) (context.Context, trace.Span) {
	return db.tracer.Start(ctx, spanName,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBStatementKey.String(query),
			semconv.DBSystemKey.String(db.GetDatabaseEngine()),
			netPeerAddressKey.String(db.cfg.Address),
			semconv.PeerServiceKey.String(fmt.Sprintf("%s[%s(%s)]", db.cfg.DatabaseName, db.GetDatabaseEngine(), db.cfg.Address)),
		))
}

func (db *Instance) QueryRowContext(ctx context.Context, query string, args ...interface{}) *Row {
	ctx, span := startSpan(ctx, db, "QueryRowContext", query)
	defer span.End()

	if len(args) > 0 {
		var err error
		query, args, err = sqlx.In(query, args...)
		if err != nil {
			return nil
		}
	}

	finalQuery := db.db.Rebind(query)

	if isTransaction(ctx) {
		return &Row{err: fmt.Errorf("tried to use database with a transaction context")}
	}
	return &Row{row: db.db.QueryRowxContext(ctx, finalQuery, args...)}
}

func (db *Instance) QueryContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error) {
	ctx, span := startSpan(ctx, db, "QueryContext", query)
	defer span.End()

	if len(args) > 0 {
		var err error
		query, args, err = sqlx.In(query, args...)
		if err != nil {
			return nil, err
		}
	}

	finalQuery := db.db.Rebind(query)

	if isTransaction(ctx) {
		return nil, fmt.Errorf("tried to use database with a transaction context")
	}
	return db.db.QueryxContext(ctx, finalQuery, args...)
}

func (db *Instance) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	ctx, span := startSpan(ctx, db, "ExecContext", query)
	defer span.End()

	if len(args) > 0 {
		var err error
		query, args, err = sqlx.In(query, args...)
		if err != nil {
			return nil, err
		}
	}

	finalQuery := db.db.Rebind(query)

	if isTransaction(ctx) {
		return nil, fmt.Errorf("tried to use database with a transaction context")
	}
	return db.db.ExecContext(ctx, finalQuery, args...)
}

func (db *Instance) ExecAndReturnId(ctx context.Context, query string, args ...interface{}) (int64, error) {
	return ExecAndReturnId(db, ctx, query, args...)
}

func (db *Instance) GenerateLimitCondition(pageSize int64, pageNumber int64, orderByFieldNameForSqlSvr string) (string, error) {
	switch strings.ToLower(db.GetDatabaseEngine()) {
	case "sqlite":
		fallthrough
	case "mysql":
		fallthrough
	case "postgres":
		return fmt.Sprintf(" LIMIT %d ", pageSize), nil
	default:
		return "", ErrDatabaseEngineNotSupported
	}
}

func (db *Instance) GenerateLimitAndOrderCondition(pageSize int64, pageNumber int64, orderByFieldName string, isDesc bool) (string, error) {
	maybeDescString := ""
	if isDesc {
		maybeDescString = " DESC "
	}
	switch strings.ToLower(db.GetDatabaseEngine()) {
	case "sqlite":
		fallthrough
	case "mysql":
		fallthrough
	case "postgres":
		return fmt.Sprintf(" ORDER BY %s%s LIMIT %d, %d", orderByFieldName, maybeDescString, pageNumber, pageSize), nil
	default:
		return "", ErrDatabaseEngineNotSupported
	}
}

// ctxExecQuerier is a sealed interface that propagates either a Tx or DB into ExecAndReturnId helper function
type ctxExecQuerier interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sqlx.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *Row
	GetDatabaseEngine() string
}

// ExecAndReturnId - query string must contain "VALUES" (uppercase) keyword
func ExecAndReturnId(ceq ctxExecQuerier, ctx context.Context, query string, args ...interface{}) (int64, error) {
	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return 0, err
	}

	switch strings.ToLower(ceq.GetDatabaseEngine()) {
	case "sqlite":
		fallthrough
	case "mysql":
		fallthrough
	case "postgres":
		res, err := ceq.ExecContext(ctx, query, args...)
		if err != nil {
			log.Printf("Failed to save to database - %s", err)
			return 0, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			log.Printf("failed to get last inserted id", err)
			return 0, err
		}
		return id, nil
	default:
		return 0, ErrDatabaseEngineNotSupported
	}
}
