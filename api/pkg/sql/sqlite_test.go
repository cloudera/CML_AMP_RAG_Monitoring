package lsql

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSqliteInitialize(t *testing.T) {
	db, err := initializeTest(t)
	assert.Nil(t, err)
	assert.NotNil(t, db)
	_, err = db.ExecContext(context.Background(), "create table t(i);")
	assert.Nil(t, err)
}
