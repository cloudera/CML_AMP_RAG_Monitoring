#!/usr/bin/env bash
set -eo pipefail

# check if uv command is present and if not, install it
if ! command -v uv &> /dev/null; then
    echo "uv command not found, installing uv"
    pip install uv
fi

cd ragmon
uv sync

MLFLOW_VERSION="2.16.2"

echo "Installing MLflow $MLFLOW_VERSION"
uv pip install --no-cache-dir protobuf sqlparse "mlflow==$MLFLOW_VERSION"
