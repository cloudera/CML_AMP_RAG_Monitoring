import json
import requests
import streamlit as st

from data_types import (
    MLFlowExperimentRequest,
    MLFlowStoreIdentifier,
    MLFlowStoreMetricRequest,
)


@st.cache_data(show_spinner=True)
def get_experiments():
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
        timeout=60,
    )
    response_json = response.json()
    if not response_json:
        return []
    return response_json


@st.cache_data(show_spinner=True)
def get_runs(request: MLFlowExperimentRequest):
    """
    Fetches the list of runs for a given experiment ID from the MLflow store.

    Args:
        request (MLFlowExperimentRequest): The request object containing the experiment ID.

    Returns:
        list: A list of runs for the given experiment ID. Returns an empty list if no runs are found or if the response is empty.

    Raises:
        requests.exceptions.RequestException: If there is an issue with the HTTP request.
    """
    uri = "http://localhost:3000/runs/list"
    response = requests.post(
        url=uri,
        data=request.json(),
        headers={
            "Content-Type": "application/json",
        },
        timeout=60,
    )
    response_json = response.json()
    if not response_json:
        return []
    return response_json


@st.cache_data(show_spinner=True)
def get_metric_names(request: MLFlowExperimentRequest):
    """
    Fetches a list of metric names from the MLflow store.

    Args:
        request (MLFlowExperimentRequest): The request object containing the experiment ID.

    Returns:
        list: A list of metric names retrieved from the response. If the response is not successful, returns an empty list.
    """
    uri = "http://localhost:3000/metrics/names"
    response = requests.get(
        url=uri,
        params=request.dict(),
        headers={
            "Accept": "application/json",
        },
        timeout=60,
    )
    response_json = response.json()
    if not response_json:
        return []
    return response_json


@st.cache_data(show_spinner=True)
def get_parameters(request: MLFlowStoreIdentifier):
    """
    Fetches the parameters for a given experiment run ID from the MLflow store.

    Args:
        request (MLFlowStoreIdentifier): The request object containing the experiment run ID.

    Returns:
        dict: A dictionary containing the parameters for the given experiment run ID. If the response is empty, returns an empty dictionary.
    """
    uri = "http://localhost:3000/runs/parameters"
    response = requests.get(
        url=uri,
        params=request.dict(),
        headers={
            "Accept": "application/json",
        },
        timeout=60,
    )
    response_json = response.json()
    if not response_json:
        return {}
    return response_json


def merge_jsons(*dicts):
    """
    Merges multiple dictionaries into a single dictionary.

    Args:
        *dicts: A variable number of dictionaries to merge.

    Returns:
        dict: A dictionary containing the merged key-value pairs from all input dictionaries.
    """
    merged = {}

    for d in dicts:
        for key, value in d.items():
            if key in merged:
                if merged[key] != value:
                    if not isinstance(merged[key], list):
                        merged[key] = [merged[key]]
                    if value not in merged[key]:
                        merged[key].append(value)
            else:
                merged[key] = value

    return merged


@st.cache_data(show_spinner=True)
# pull data from metric store
def get_metrics(
    request: MLFlowStoreMetricRequest,
):
    """
    Sends a POST request to the MLflow store to retrieve metrics.

    Args:
        request (MLFlowStoreMetricRequest): The request object containing the data to be sent in the POST request.

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
        timeout=60,
    )
    # if response is not successful, return empty list
    if not response.ok:
        return []
    return response.json()


@st.cache_data(show_spinner=True)
def get_json(
    request: MLFlowStoreMetricRequest,
):
    """
    Fetches JSON data from the MLflow store.

    Args:
        request (MLFlowStoreMetricRequest): The request object containing the data to be sent in the POST request.

    Returns:
        list: A list of JSON data retrieved from the response. If the response is not successful, returns an empty list.
    """
    json_dicts = get_metrics(request)
    for json_dict in json_dicts:
        new_json_list = []
        if json_dict["value"]["metricType"] == "text":
            json_dict["value"]["stringValue"] = json.loads(
                json_dict["value"]["stringValue"]
            )
        columns = json_dict["value"]["stringValue"]["columns"]
        for data in json_dict["value"]["stringValue"]["data"]:
            new_json_list.append(dict(zip(columns, data)))
        merged_json = merge_jsons(*new_json_list)
        json_dict["value"] = merged_json
    return json_dicts
