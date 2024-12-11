#!/usr/bin/env bash
set -eo pipefail

pip install uv

cd ragmon
uv sync

MLFLOW_VERSION="2.16.2"

echo "Installing MLflow $MLFLOW_VERSION"
uv pip install --no-cache-dir protobuf sqlparse "mlflow==$MLFLOW_VERSION"
