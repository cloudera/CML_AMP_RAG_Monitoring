package lsql

import (
	"fmt"
)

var (
	ErrDatabaseEngineNotSupported            = fmt.Errorf("database engine not supported")
	ErrDatabaseParameterCountMismatch        = fmt.Errorf("query contains different number of parameters than provided")
	ErrUpsertFailure                         = fmt.Errorf("failed to insert or update row in database")
	ErrOrderByRequired                       = fmt.Errorf("sqlserver requires the ORDER BY value to be set")
	MySQLDuplicateRowErrno            uint16 = 1062
)
