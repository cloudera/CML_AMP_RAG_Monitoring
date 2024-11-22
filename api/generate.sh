#!/bin/bash

set -eo pipefail

GITROOT=$(git rev-parse --show-toplevel)

go install github.com/go-swagger/go-swagger/cmd/swagger@v0.28.0

rm -f swagger.yaml
cat schema/base.yaml > swagger.yaml
echo "paths:" >> swagger.yaml
for p in schema/paths/*.yaml
do
    cat $p | sed 's,^,  ,' >> swagger.yaml
    echo '' >> swagger.yaml
done
echo "definitions:" >> swagger.yaml
for d in schema/definitions/*.yaml
do
    cat $d | sed 's,^,  ,' >> swagger.yaml
    echo '' >> swagger.yaml
#    t=$(echo $d | xargs basename | sed 's,\.yaml$,,')
#    grep -r "'#/definitions/$t'" schema > /dev/null
#    if [ $? -eq 1 ]
#    then
#        echo "WARN: definition $t is not referenced anywhere"
#    fi
done
rm -rf {client,models,restapi}
swagger generate server -T=$GITROOT/api/pkg/swagger/templates -t . -f swagger.yaml --exclude-main
swagger generate client -T=$GITROOT/api/pkg/swagger/templates -t . -f swagger.yaml

cd $GITROOT/api/internal/migrations
./generate.sh
cd $GITROOT/api

cd $GITROOT/api/cmd
wire
cd $GITROOT/api
go fmt ./...
