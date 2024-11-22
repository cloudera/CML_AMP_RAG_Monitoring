#!/usr/bin/env bash
set -eo pipefail

cd api
cd cmd
GOOS=linux GOARCH=amd64 go build -o ../api .
