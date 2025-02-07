#!/usr/bin/env bash
set -eo pipefail

# check if uv command is present and if not, install it
if ! command -v uv &> /dev/null; then
    echo "uv command not found, installing uv"
    pip install uv
fi

cd ragmon
uv sync
