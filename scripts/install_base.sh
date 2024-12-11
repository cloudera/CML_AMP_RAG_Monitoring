#!/usr/bin/env bash
set -eo pipefail

mkdir -p ~/.ssh
ssh-keyscan -t ecdsa github.com >> ~/.ssh/known_hosts

scripts/install_sqlite.sh
scripts/install_qdrant.sh
scripts/install_golang.sh

