"""
Run this test script using the following command:

pytest -s test_llm_eval.py

This script tests the logging of GenAI-specific metrics using the mlflow.metrics.genai module.
The script logs the metrics for a mock evaluation run and fetches them using the MLflow tracking API.
The test checks if the expected metrics are logged and fetched successfully.

The script uses the following fixtures:
1. experiment: Sets up an MLflow experiment for logging metrics.
2. log_metrics: Logs the GenAI-specific metrics for a mock evaluation run and returns the run_id.
3. fetch_metrics: Fetches the logged metrics using the MLflow tracking API.

The test_metrics_logged function checks if the expected metrics are logged and fetched successfully.
If the expected metrics are not found, the test raises an assertion error with a detailed message.

You can modify the inputs, outputs, and context for the mock evaluation run.
You can change the model to use as judge `model` in the log_metrics fixture.
You can test it on your local mlflow server by:
1. Uncomment the mlflow.set_tracking_uri line in the experiment fixture and add tracking URI.
2. Adding the tracking_uri parameter to the MlflowClient in the fetch_metrics function
3. 

"""

import time
import warnings
import pytest
import mlflow
from mlflow.metrics.genai import faithfulness, answer_relevance, relevance
from mlflow.tracking import MlflowClient
import pandas as pd

# Suppress warnings
warnings.simplefilter("ignore", category=DeprecationWarning)
warnings.simplefilter("ignore", category=PendingDeprecationWarning)

# Mock inputs, outputs, and context for 5 runs
inputs = [
    "What is the capital of France?",
    "Who wrote 'To Kill a Mockingbird'?",
    "What is the chemical symbol for water?",
    "How many continents are there on Earth?",
    "What is the speed of light?",
]

predictions = [
    "The capital of France is Paris.",
    "'To Kill a Mockingbird' was written by Harper Lee.",
    "The chemical symbol for water is H2O.",
    "There are seven continents on Earth.",
    "The speed of light is approximately 299,792 kilometers per second.",
]

contexts = [
    "France is a country in Europe. Its capital city is known for its art, fashion, and culture.",
    "'To Kill a Mockingbird' is a novel published in 1960. It was written by an American author.",
    "Water is a molecule composed of two hydrogen atoms and one oxygen atom.",
    "Earth has seven continents: Africa, Antarctica, Asia, Europe, North America, Australia, and South America.",
    "Light travels at a constant speed in a vacuum, which is a fundamental constant of nature.",
]

eval_df = pd.DataFrame(
    {
        "inputs": inputs,
        "predictions": predictions,
        "context": contexts,
    }
)


@pytest.fixture(scope="module")
def experiment():
    # Set tracking URI if needed by uncommenting
    # mlflow.set_tracking_uri("<tracking_uri>")

    # Set up MLflow experiment
    experiment_name = "/mock_metrics_genai/llm_eval"
    mlflow.set_experiment(experiment_name)

    yield experiment_name


@pytest.fixture
def log_metrics(experiment):
    # Helper function to log the metrics and return the run_id
    with mlflow.start_run() as run:
        # Define the model to evaluate (change if needed)
        model = "bedrock:/cohere.command-r-plus-v1:0"

        faithfulness_metric = faithfulness(
            model=model,
        )
        relevance_metric = relevance(
            model=model,
        )
        answer_relevance_metric = answer_relevance(
            model=model,
        )
        results = mlflow.evaluate(
            data=eval_df,
            model_type="question-answering",
            evaluators="default",
            predictions="predictions",
            extra_metrics=[
                faithfulness_metric,
                relevance_metric,
                answer_relevance_metric,
            ],
        )

        print(f"Evaluation metrics for run {run.info.run_id}:")
        eval_table = results.tables["eval_results_table"]
        print(f"\n{eval_table}")

        # Return the run_id for fetching metrics later
        return run.info.run_id


def fetch_metrics(run_id):
    # Fetch the logged metrics using MLflow's tracking API
    # if needed add tracking_uri="<tracking_uri>"" to MlflowClient
    client = MlflowClient()
    run_data = client.get_run(run_id).data
    return run_data.metrics


def test_metrics_logged(log_metrics):
    run_id = log_metrics
    print("Simulating a delay before fetching metrics...")
    time.sleep(5)  # Simulate some delay between logging and fetching metrics
    print("Fetching metrics...")
    # Fetch the metrics from the server
    metrics = fetch_metrics(run_id)

    # Detailed messages for assertion failures
    missing_metrics = []

    # Check for each expected metric
    if (
        "faithfulness/v1/mean" not in metrics
        and "faithfulness/v1/variance" not in metrics
    ):
        missing_metrics.append("faithfulness")
    if (
        "answer_relevance/v1/mean" not in metrics
        and "answer_relevance/v1/variance" not in metrics
    ):
        missing_metrics.append("answer_relevance")
    if "relevance/v1/mean" not in metrics and "relevance/v1/variance" not in metrics:
        missing_metrics.append("relevance")

    # If there are missing metrics, raise an assertion error with a detailed message
    if missing_metrics:
        pytest.fail(
            f"Missing the following expected metrics: {', '.join(missing_metrics)}"
        )

    # Print the fetched metrics for verification
    print("\nFetched Metrics:")
    for metric, value in metrics.items():
        print(f"{metric}: {value}")

    # Detailed success message
    print("\nAssertions passed: All expected metrics were found in the MLflow run.")
