#!/bin/bash

set -eo pipefail

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

SQLITE="sqlite"
POSTGRES="postgres"

go install github.com/jteeuwen/go-bindata/go-bindata@v3.0.7

pushd "$SCRIPT_DIR/$SQLITE"
go-bindata -pkg ${SQLITE} *.sql
popd

pushd "$SCRIPT_DIR/$POSTGRES"
go-bindata -pkg ${POSTGRES} *.sql
popd

go fmt ./...
