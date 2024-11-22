#!/usr/bin/env bash
set -eo pipefail

SQLITE_ZIP_URL=https://www.sqlite.org/2024/sqlite-tools-linux-x64-3460100.zip
SQLITE_ZIP=sqlite-tools-linux-x64-3460100.zip

mkdir -p sqlite
cd sqlite

rm -f ${SQLITE_ZIP}
wget --no-verbose -O ${SQLITE_ZIP} ${SQLITE_ZIP_URL}
unzip ${SQLITE_ZIP} #&& rm ${SQLITE_ZIP}

