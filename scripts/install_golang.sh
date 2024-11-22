#!/usr/bin/env bash

GO_TGZ=go1.23.1.linux-amd64.tar.gz
GO_TGZ_URL="https://go.dev/dl/${GO_TGZ}"

## Install Go ##
rm -f ${GO_TGZ}
wget --no-verbose -O ${GO_TGZ} ${GO_TGZ_URL}
tar xzf ${GO_TGZ} && rm ${GO_TGZ}

mkdir -p /home/cdsw/.local/bin
ln -fs ~/go/bin/go /home/cdsw/.local/bin/go
ln -fs ~/go/bin/gofmt /home/cdsw/.local/bin/gofmt
