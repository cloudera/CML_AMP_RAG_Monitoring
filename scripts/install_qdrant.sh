#!/usr/bin/env bash
set -eo pipefail

QDRANT_TGZ=qdrant.tar.gz
VERSION=1.11.5
DL_URL="https://github.com/qdrant/qdrant/releases/download/v$VERSION/qdrant-x86_64-unknown-linux-musl.tar.gz"

mkdir -p qdrant 2>/dev/null
cd qdrant

## Install Qdrant ##
rm -f ${QDRANT_TGZ}
wget --no-verbose -O ${QDRANT_TGZ} ${DL_URL}
tar xzf ${QDRANT_TGZ} && rm ${QDRANT_TGZ}
