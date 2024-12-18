import json
from typing import List
import numpy as np
import pandas as pd
import requests
import streamlit as st
import plotly.graph_objects as go
from streamlit.delta_generator import DeltaGenerator

from data_types import MLFlowStoreRequest

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
    metrics_df["timestamp"] = pd.to_datetime(
        metrics_df["timestamp"], format="mixed", dayfirst=True
    )
    metrics_df = metrics_df.sort_values(by="timestamp", ascending=True)
    return metrics_df


def show_live_df_component(
    live_results_df: pd.DataFrame,
):
    if not live_results_df.empty:
        live_results_df = live_results_df.drop(columns=["response_id", "run_id"])
        live_results_df["feedback"] = live_results_df["feedback"].apply(
            lambda x: "👍" if x == 1 else "👎" if x == 0 else "🤷‍♂️"
        )
        st.write("### Detailed Logs")
        st.dataframe(live_results_df.sort_values(by="timestamp", ascending=False))


def show_i_o_component(
    input_lengths_df: pd.DataFrame,
    output_lengths_df: pd.DataFrame,
    input_kpi_placeholder: DeltaGenerator,
    output_kpi_placeholder: DeltaGenerator,
    update_timestamp: str,
):
    """
    Display input and output length KPIs and their respective time series plots.

    Parameters:
    input_lengths_df (pd.DataFrame): DataFrame containing input lengths and timestamps.
    output_lengths_df (pd.DataFrame): DataFrame containing output lengths and timestamps.
    input_kpi_placeholder (DeltaGenerator): Streamlit placeholder for input KPI metric.
    output_kpi_placeholder (DeltaGenerator): Streamlit placeholder for output KPI metric.
    update_timestamp (str): Timestamp to uniquely identify the update for Streamlit components.

    Returns:
    None
    """
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

    with st.expander(
        ":material/input:/:material/output: **I/O Overview**",
        expanded=True,
    ):
        fig_col4, fig_col5 = st.columns(2)

        input_lengths_df = input_lengths_df[["input_length", "timestamp"]]
        agg_input_df = input_lengths_df.groupby(
            pd.Grouper(key="timestamp", freq="h")  # group by hour
        )["input_length"].agg(["mean", "max", "min"])
        with fig_col4:
            st.markdown(
                "### Input Length",
                help="The average number of words in the input.",
            )
            fig = go.Figure(
                data=go.Scatter(
                    x=agg_input_df.index,
                    y=agg_input_df["mean"],
                    mode="markers",
                    marker=dict(size=5),
                    fill="tozeroy",
                    customdata=agg_input_df[["max", "min"]],
                    hovertemplate="Mean: <b>%{y:.2f}</b> Max: <b>%{customdata[0]:.2f}"
                    "</b><br>Min: <b>%{customdata[1]:.2f}</b><br>Date: %{x|%b %d, %Y}"
                    "<br>Time: %{x|%H:%M}<extra></extra>",
                )
            )
            fig.update_layout(
                xaxis_title="Date",
                yaxis_title="Mean Input Length (in words)",
                xaxis={
                    "tickformat": "%b %d, %Y",
                    "tickmode": "array",
                },
            )
            st.plotly_chart(fig, key=f"input_length_fig_{update_timestamp}")

        output_lengths_df = output_lengths_df[["output_length", "timestamp"]]
        agg_output_df = output_lengths_df.groupby(
            pd.Grouper(key="timestamp", freq="h")  # group by hour
        )["output_length"].agg(["mean", "max", "min"])
        with fig_col5:
            st.markdown(
                "### Output Length",
                help="The average number of words in the output.",
            )
            fig = go.Figure(
                data=go.Scatter(
                    x=agg_output_df.index,
                    y=agg_output_df["mean"],
                    mode="markers",
                    marker=dict(size=5),
                    fill="tozeroy",
                    customdata=agg_output_df[["max", "min"]],
                    hovertemplate="Mean: <b>%{y:.2f}</b> Max: <b>%{customdata[0]:.2f}"
                    "</b><br>Min: <b>%{customdata[1]:.2f}</b><br>Date: %{x|%b %d, %Y}"
                    "<br>Time: %{x|%H:%M}<extra></extra>",
                )
            )
            fig.update_layout(
                xaxis_title="Date",
                yaxis_title="Mean Output Length (in words)",
                xaxis={
                    "tickformat": "%b %d, %Y",
                    "tickmode": "array",
                },
            )
            st.plotly_chart(fig, key=f"output_length_fig_{update_timestamp}")


def show_feedback_component(
    feedback_df: pd.DataFrame,
    thumbs_up_placeholder: DeltaGenerator,
    thumbs_down_placeholder: DeltaGenerator,
    no_feedback_placeholder: DeltaGenerator,
    update_timestamp: str,
):
    """
    Display feedback KPI and pie chart of feedback distribution.

    Parameters:
    feedback_df (pd.DataFrame): DataFrame containing feedback and timestamps.
    feedback_placeholder (DeltaGenerator): Streamlit placeholder for feedback KPI metric.
    update_timestamp (str): Timestamp to uniquely identify the update for Streamlit components.

    Returns:
    None
    """
    if "feedback" in feedback_df:
        thumbs_up_count = feedback_df["feedback"].to_list().count(1)
        thumbs_down_count = feedback_df["feedback"].to_list().count(0)
        no_feedback_count = feedback_df["feedback"].isna().sum()

        thumbs_up_placeholder.metric(
            label="Thumbs Up :material/thumb_up:",
            help="The number of thumbs up received.",
            value=thumbs_up_count,
        )

        thumbs_down_placeholder.metric(
            label="Thumbs Down :material/thumb_down:",
            help="The number of thumbs down received.",
            value=thumbs_down_count,
        )

        no_feedback_placeholder.metric(
            label="No Feedback :material/indeterminate_question_box:",
            help="The number of no feedback received.",
            value=no_feedback_count,
        )

        with st.expander("# :material/feedback: **Feedback Overview**", expanded=True):
            st.markdown(
                "### Feedback Received",
                help="Feedback received from users.",
            )
            fig = go.Figure(
                data=go.Pie(
                    labels=["Thumbs Up", "Thumbs Down", "No Feedback"],
                    values=[
                        thumbs_up_count,
                        thumbs_down_count,
                        no_feedback_count,
                    ],
                    hole=0.5,
                    hovertemplate="%{label}: <b>%{value}</b><extra></extra>",
                )
            )
            st.plotly_chart(fig, key=f"feedback_fig_{update_timestamp}")


def show_numeric_metric_kpi(
    metric_key: str,
    metrics_df: pd.DataFrame,
    kpi_placeholder: DeltaGenerator,
    label: str,
    tooltip: str,
):
    """
    Display numeric metric KPIs.

    Parameters:
    metrics_df (pd.DataFrame): DataFrame containing numeric metrics and timestamps.
    kpi_placeholder (DeltaGenerator): Streamlit placeholder for numeric metric KPI.

    Returns:
    None
    """
    if metric_key in metrics_df.columns:
        avg_metric = np.mean(metrics_df[metric_key])
        metric_scores = metrics_df[metrics_df[metric_key].notna()][metric_key].to_list()
        # fill in those three columns with respective metrics or KPIs
        kpi_placeholder.metric(
            label=label,
            help=tooltip,
            value=round(avg_metric, 2),
            delta=round(
                (
                    avg_metric - np.mean(metric_scores[:-1])
                    if len(metric_scores) > 1
                    else 0
                ),
                2,
            ),
        )


def show_pie_chart_component(
    metric_key: str,
    metrics_df: pd.DataFrame,
    title: str,
    tooltip: str,
    labels: List[str],
    update_timestamp: str,
):
    """
    Displays a pie chart component in a Streamlit app.

    Parameters:
    metric_key (str): The key to identify the metric in the DataFrame.
    metrics_df (pd.DataFrame): The DataFrame containing the metrics data.
    update_timestamp (str): A timestamp string to ensure the chart is updated.
    title (str): The title of the pie chart.
    tooltip (str): The tooltip text for the pie chart title.
    labels (List[str]): The labels for the pie chart slices.

    Returns:
    None
    """
    if metric_key in metrics_df:
        st.markdown(f"### {title}", help=tooltip)
        fig = go.Figure(
            data=go.Pie(
                labels=labels,
                values=[
                    metrics_df[metric_key].to_list().count(1),
                    metrics_df[metric_key].to_list().count(0),
                    metrics_df[metric_key].isna().sum(),
                ],
                hole=0.5,
                hovertemplate="%{label}: <b>%{value}</b><extra></extra>",
            )
        )
        st.plotly_chart(fig, key=f"{metric_key}_fig_{update_timestamp}")


def show_time_series_component(
    metric_key: str,
    metrics_df: pd.DataFrame,
    title: str,
    tooltip: str,
    update_timestamp: str,
    frequency: str = "h",
):
    """
    Displays a time series component in a Streamlit app.

    Parameters:
    metric_key (str): The key to identify the metric in the DataFrame.
    metrics_df (pd.DataFrame): The DataFrame containing the metrics data.
    update_timestamp (str): A timestamp string to ensure the chart is updated.
    title (str): The title of the time series plot.
    tooltip (str): The tooltip text for the time series plot title.

    Returns:
    None
    """
    if metric_key in metrics_df:
        st.markdown(f"### {title}", help=tooltip)
        metrics_df = metrics_df[[metric_key, "timestamp"]]
        agg_df = metrics_df.groupby(
            pd.Grouper(key="timestamp", freq=frequency)  # group by frequency
        )[metric_key].agg(["mean", "max", "min"])
        fig = go.Figure(
            data=go.Scatter(
                x=agg_df.index,
                y=agg_df["mean"],
                mode="markers",
                marker=dict(size=5),
                fill="tozeroy",
                customdata=agg_df[["max", "min"]],
                hovertemplate="Mean: <b>%{y:.2f}</b> Max: <b>%{customdata[0]:.2f}"
                "</b><br>Min: <b>%{customdata[1]:.2f}</b><br>Date: %{x|%b %d, %Y}"
                "<br>Time: %{x|%H:%M}<extra></extra>",
            )
        )
        fig.update_layout(
            xaxis_title="Date",
            yaxis_title=f"Mean {title}",
            xaxis={
                "tickformat": "%b %d, %Y",
                "tickmode": "array",
            },
        )
        st.plotly_chart(fig, key=f"{metric_key}_fig_{update_timestamp}")
