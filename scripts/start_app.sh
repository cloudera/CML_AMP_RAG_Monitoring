#!/usr/bin/env bash

export FASTAPI_PORT=8200
export SQL_DB_PATH="monitoring.db"

# if AWS_DEFAULT_REGION isn't set, set to $AWS_REGION
if [[ -z "$AWS_DEFAULT_REGION" ]]; then
    export AWS_DEFAULT_REGION=$AWS_REGION
fi

set -eo pipefail

cleanup() {
    # kill all processes whose parent is this process
    pkill -P $$
}

for sig in INT QUIT HUP TERM; do
  trap "
    cleanup
    trap - $sig EXIT
    kill -s $sig "'"$$"' "$sig"
done
trap cleanup EXIT

echo "Starting Qdrant"
# Start Qdrant
nohup ./qdrant/qdrant >~/qdrant.log &

echo "Starting Metric API"
# Start up the Go api
SQL_DB_NAME=rag_metrics SQL_DB_ADDRESS=rag_metrics nohup ./api/api &>~/api.log &

cd ragmon

echo "Starting MLflow"
# Start MLflow
nohup uv run mlflow server --backend-store-uri "sqlite:///$SQL_DB_PATH" &>../mlflow.log &

echo "Starting REST API"
# Start FastAPI
nohup uv run fastapi run --host 127.0.0.1 --port $FASTAPI_PORT main.py &>../fastapi.log &

echo "Waiting for REST API"
# Wait for FastAPI to boot and settle
sleep 60

# Run the pre-population script
echo "Pre-populating data"
nohup uv run populate_and_simulate_qa.py >../populate_and_simulate_qa.log 

ADDRESS=${ADDRESS:-127.0.0.1}

# Start Streamlit
echo "Starting UI"
uv run streamlit run app.py --server.address $ADDRESS --server.port $CDSW_APP_PORT
