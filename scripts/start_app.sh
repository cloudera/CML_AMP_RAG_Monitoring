#!/usr/bin/env bash

export FASTAPI_PORT=8200
export SQL_DB_PATH="monitoring.db"

# Set AWS_DEFAULT_REGION to the value of AWS_REGION
export AWS_DEFAULT_REGION=$AWS_REGION

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

# Start Qdrant
nohup ~/qdrant/qdrant >~/qdrant.log &

# Start MLflow
nohup uv run --no-project mlflow server --backend-store-uri "sqlite:///$SQL_DB_PATH" &>~/mlflow.log &

# Start up the Go api
SQL_DB_NAME=rag_metrics SQL_DB_ADDRESS=rag_metrics nohup ./api/api &>~/api.log &

# Start FastAPI
nohup uv run --no-project fastapi run --host 127.0.0.1 --port $FASTAPI_PORT ~/service/main.py >~/fastapi.log &

# Run the pre-population script
nohup uv run --no-project ~/scripts/populate_and_simulate_qa.py >~/populate_and_simulate_qa.log 

ADDRESS=${ADDRESS:-127.0.0.1}

# Start Streamlit
uv run --no-project streamlit run ~/st_app/app.py --server.address $ADDRESS --server.port $CDSW_APP_PORT
