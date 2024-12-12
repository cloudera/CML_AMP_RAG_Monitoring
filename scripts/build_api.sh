#!/usr/bin/env bash
set -eo pipefail

export PATH=$PATH:/home/cdsw/.local/bin/
cd api
cd cmd
GOOS=linux GOARCH=amd64 go build -o ../api .
