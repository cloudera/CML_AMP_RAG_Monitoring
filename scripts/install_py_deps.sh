#!/usr/bin/env bash
set -eo pipefail

uv pip install --no-cache-dir -r service/requirements.txt
uv pip install --no-cache-dir -r st_app/requirements.txt
