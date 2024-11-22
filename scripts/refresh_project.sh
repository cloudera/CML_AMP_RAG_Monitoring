#!/usr/bin/env bash
set -eo pipefail

current_branch=$(git rev-parse --abbrev-ref HEAD)

git reset --hard origin/$current_branch

echo "Building API"
scripts/build_api.sh

echo "Installing Python dependencies"
scripts/install_py_deps.sh