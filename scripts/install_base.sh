#!/usr/bin/env bash
set -eo pipefail

mkdir -p ~/.ssh
ssh-keyscan -t ecdsa github.com >> ~/.ssh/known_hosts

scripts/install_sqlite.sh
scripts/install_qdrant.sh
scripts/install_golang.sh

MLFLOW_VERSION="2.16.2"

pip install uv
uv venv

if ! command -v mlflow &> /dev/null; then
  echo "Installing MLflow $MLFLOW_VERSION"
  uv pip install --no-cache-dir protobuf sqlparse "mlflow==$MLFLOW_VERSION"
fi
