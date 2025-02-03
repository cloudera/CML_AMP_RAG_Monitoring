package lsql

import (
	"fmt"
)

var (
	ErrDatabaseEngineNotSupported            = fmt.Errorf("database engine not supported")
	ErrDatabaseParameterCountMismatch        = fmt.Errorf("query contains different number of parameters than provided")
	ErrUpsertFailure                         = fmt.Errorf("failed to insert or update row in database")
	MySQLDuplicateRowErrno            uint16 = 1062
)
