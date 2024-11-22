#!/usr/bin/env bash

set -eo pipefail

poetry export --format requirements.txt > st_app/requirements.txt
cp st_app/requirements.txt service/app/requirements.txt
