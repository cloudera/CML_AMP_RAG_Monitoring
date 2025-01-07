#!/usr/bin/env bash
set -eo pipefail

# if PATH doesn't already include /home/cdsw/.local/bin, add it
if [[ ! $PATH == *"/home/cdsw/.local/bin"* ]]; then
    export PATH="/home/cdsw/.local/bin:$PATH"
fi

cd api
cd cmd
GOOS=linux GOARCH=amd64 go build -o ../api .
