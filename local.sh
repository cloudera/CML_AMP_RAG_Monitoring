#!/usr/bin/env bash
set -eo pipefail

CDSW_APP_PORT=${CDSW_APP_PORT:-8100}

if [ -z "$CDSW_API_URL" ] ; then
  echo "CDSW_API_URL"
  exit 1
fi

if [ -z "$CDSW_PROJECT_NUM" ] ; then
  echo "CDSW_PROJECT_NUM must be set"
  exit 1
fi

if [ -z "$CDSW_API_KEY" ] ; then
  echo "CDSW_API_KEY must be set"
  exit 1
fi

if [ -z "$AWS_REGION" ] && [ -z "$CAII_DOMAIN" ]; then
  echo "Either AWS_REGION or CAII_DOMAIN must be set"
  exit 1
fi

if [ -n "$AWS_REGION" ]; then
  if [ -z "$AWS_ACCESS_KEY_ID" ] || [ -z "$AWS_SECRET_ACCESS_KEY" ]; then
    echo "AWS_REGION is set, so AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set"
    exit 1
  else
    DOCKER_CMD_ENV="-e AWS_REGION=$AWS_REGION -e AWS_ACCESS_KEY_ID=$AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY=$AWS_SECRET_ACCESS_KEY"
  fi
fi

if [ -n "$CAII_DOMAIN" ]; then
  if [ -z "$CAII_INFERENCE_ENDPOINT_NAME" ] || [ -z "$CAII_EMBEDDING_ENDPOINT_NAME" ]; then
    echo "CAII_DOMAIN is set, so CAII_INFERENCE_ENDPOINT_NAME and CAII_EMBEDDING_ENDPOINT_NAME must be set"
    exit 1
  else
    DOCKER_CMD_ENV="-e CAII_DOMAIN=$CAII_DOMAIN -e CAII_INFERENCE_ENDPOINT_NAME=$CAII_INFERENCE_ENDPOINT_NAME -e CAII_EMBEDDING_ENDPOINT_NAME=$CAII_EMBEDDING_ENDPOINT_NAME"
  fi
fi

docker build --platform=linux/amd64 -t ragmon:latest .
docker run --platform=linux/amd64 -it --rm "$DOCKER_CMD_ENV" -e LOCAL=true -e ADDRESS=0.0.0.0 -e CDSW_API_URL="$CDSW_API_URL" -e CDSW_PROJECT_NUM="$CDSW_PROJECT_NUM" -e CDSW_APP_PORT="$CDSW_APP_PORT" -p "$CDSW_APP_PORT":"$CDSW_APP_PORT" ragmon:latest
