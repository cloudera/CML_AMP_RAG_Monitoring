import json
from typing import List
import numpy as np
import pandas as pd
import requests
from streamlit.delta_generator import DeltaGenerator

from ..data_types import MLFlowStoreRequest

table_cols_to_show = [
    "response_id",
    "run_id",
    "timestamp",
    "input",
    "output",
    "contexts",
    "input_length",
    "output_length",
    # "feedback_str"
]


def get_experiment_ids():
    """
    Fetches a list of unique experiment IDs from a MLFLow store.

    Sends a GET request to the local server at "http://localhost:3000/experiments"
    to retrieve experiment IDs. The response is expected to be in JSON format.
    If the response is empty, an empty list is returned. Otherwise, a list of
    unique experiment IDs is returned.

    Returns:
        list: A list of unique experiment IDs. If the response is empty, an empty
        list is returned.
    """
    uri = "http://localhost:3000/experiments"
    response = requests.get(
        url=uri,
        headers={
            "Content-Type": "application/json",
        },
        timeout=10,
    )
    response_json = response.json()
    if not response_json:
        return []
    return list(set(response_json))


def get_runs(experiment_id: str):
    """
    Fetches the list of runs for a given experiment ID from the MLflow store.

    Args:
        experiment_id (str): The ID of the experiment for which to fetch the runs.

    Returns:
        list: A list of runs for the given experiment ID. Returns an empty list if no runs are found or if the response is empty.

    Raises:
        requests.exceptions.RequestException: If there is an issue with the HTTP request.
    """
    uri = "http://localhost:3000/runs/list"
    response = requests.post(
        url=uri,
        json={"experiment_id": experiment_id},
        headers={
            "Content-Type": "application/json",
        },
        timeout=10,
    )
    response_json = response.json()
    if not response_json:
        return []
    return response_json


def parse_live_results_table(
    table_request: MLFlowStoreRequest,
    table_cols_to_show: List[str] = table_cols_to_show,
):
    """
    Parses the live results table from the given MLFlowStoreRequest.

    Args:
        table_request (MLFlowStoreRequest): The request object containing the necessary parameters to fetch metrics.

    Returns:
        pd.DataFrame: A DataFrame containing the parsed results with columns specified in `table_cols_to_show`.

    The function performs the following steps:
    1. Fetches metrics using the `get_metrics` function.
    2. Iterates through the results and constructs rows of data.
    3. Processes the `source_nodes` field to format its content.
    4. Adds `run_id` and `timestamp` to each row.
    5. Creates a DataFrame from the rows.
    6. Aggregates rows by `response_id` and concatenates `source_nodes` content.
    7. Optionally processes `feedback_str` (currently commented out).
    8. Returns a sorted DataFrame with the specified columns.
    """
    results = get_metrics(table_request)
    if results == []:
        return pd.DataFrame(columns=table_cols_to_show)
    rows = []
    for result in results:
        result_dict = json.loads(result["value"]["stringValue"])
        run_id = result["experiment_run_id"]
        columns = result_dict["columns"]
        for data in result_dict["data"]:
            if None in data:
                continue
            row = {col: val for col, val in zip(columns, data)}
            if row["source_nodes"] is not None:
                row["source_nodes"] = (
                    "filename: "
                    + row["source_nodes"]["source_file_name"]
                    + "\n===============================\n"
                    + row["source_nodes"]["content"]
                    + "\n===============================\n"
                    + f"Score: {row['source_nodes']['score']}"
                    + "\n===============================\n"
                )
            row["run_id"] = run_id
            row["timestamp"] = result["ts"]
            rows.append(row)
    result_df = pd.DataFrame(rows)
    response_ids = result_df["response_id"].unique().tolist()
    new_rows = []
    for response_id in response_ids:
        row = result_df[result_df["response_id"] == response_id].iloc[0]
        source_nodes = result_df[result_df["response_id"] == response_id][
            "source_nodes"
        ].tolist()
        new_source_nodes_col = "\n===============================\n".join(source_nodes)
        row["contexts"] = new_source_nodes_col

        ## TODO : Add feedback_str after logging is fixed

        # if "feedback_str" in row:
        #     new_feedback_col = [
        #         x for x in result_df[result_df["run_id"] == run_id]["feedback_str"]
        #     ]
        #     new_feedback_col = list(
        #         map(str, filter(lambda x: not math.isnan(x), new_feedback_col))
        #     )
        #     new_feedback_str = " ".join(new_feedback_col)
        #     row["feedback_str"] = new_feedback_str
        # else:
        #     row["feedback_str"] = ""

        new_rows.append(row)

    new_df = pd.DataFrame(new_rows)
    result_df = new_df[table_cols_to_show]
    result_df["timestamp"] = pd.to_datetime(
        result_df["timestamp"], format="mixed", dayfirst=True
    )
    result_df = result_df.sort_values(by="timestamp", ascending=True)
    return result_df


# pull data from metric store
def get_metrics(
    request: MLFlowStoreRequest,
):
    """
    Sends a POST request to the MLflow store to retrieve metrics.

    Args:
        request (MLFlowStoreRequest): The request object containing the data to be sent in the POST request.

    Returns:
        list: A list of metrics retrieved from the response. If the response is not successful, returns an empty list.
    """
    uri = "http://localhost:3000/metrics/list"
    response = requests.post(
        url=uri,
        data=request.json(),
        headers={
            "Content-Type": "application/json",
        },
        timeout=10,
    )
    # if response is not successful, return empty list
    if not response.ok:
        return []
    return response.json()


def get_numeric_metrics_df(request: MLFlowStoreRequest):
    """
    Retrieve numeric metrics from MLFlow store and return them as a DataFrame.

    Args:
        request (MLFlowStoreRequest): The request object containing parameters to fetch metrics.

    Returns:
        pd.DataFrame: A DataFrame containing the following columns:
            - run_id (str): The experiment run ID.
            - thumbs_up (float): The numeric value of the metric, or 0 if not available.
            - timestamp (int): The timestamp of the metric.
    """
    metric_name = request.metric_names[0]
    metrics_response = get_metrics(request)
    if metrics_response != []:
        metric_response_ids = [x["experiment_run_id"] for x in metrics_response]
        metric_scores = [
            x["value"]["numericValue"] if "numericValue" in x["value"] else 0
            for x in metrics_response
        ]
        metrics_ts = [x["ts"] for x in metrics_response]
        metrics_df = pd.DataFrame(
            {
                "run_id": metric_response_ids,
                metric_name: metric_scores,
                "timestamp": metrics_ts,
            }
        )
    else:
        metrics_df = pd.DataFrame(columns=["run_id", metric_name, "timestamp"])
    return metrics_df


def show_i_o_component(
    input_lengths_df: pd.DataFrame,
    output_lengths_df: pd.DataFrame,
    input_kpi_placeholder: DeltaGenerator,
    output_kpi_placeholder: DeltaGenerator,
):
    # Show input and output length KPIs
    if "input_length" in input_lengths_df.columns:
        avg_input_length = np.mean(input_lengths_df["input_length"])
        input_lengths = input_lengths_df["input_length"].to_list()
        input_kpi_placeholder.metric(
            label="Input Length :material/input:",
            help="The average number of words in the input.",
            value=round(avg_input_length, 2),
            delta=round(
                (
                    avg_input_length - np.mean(input_lengths[:-1])
                    if len(input_lengths) > 1
                    else 0
                ),
                2,
            ),
        )

    if "output_length" in output_lengths_df.columns:
        avg_output_length = np.mean(output_lengths_df["output_length"])
        output_lengths = output_lengths_df["output_length"].to_list()
        output_kpi_placeholder.metric(
            label="Output Length :material/output:",
            help="The average number of words in the output.",
            value=round(avg_output_length, 2),
            delta=round(
                (
                    avg_output_length - np.mean(output_lengths[:-1])
                    if len(output_lengths) > 1
                    else 0
                ),
                2,
            ),
        )
