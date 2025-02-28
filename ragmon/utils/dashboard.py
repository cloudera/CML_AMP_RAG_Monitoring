from typing import Any, Dict, List, Optional
from functools import reduce
import numpy as np
import pandas as pd
import streamlit as st
import plotly.graph_objects as go
from streamlit.delta_generator import DeltaGenerator
from wordcloud import WordCloud
from streamlit_extras.dataframe_explorer import dataframe_explorer

from data_types import (
    MLFlowStoreMetricRequest,
    MLFlowStoreIdentifier,
)
from .metric_store import get_metrics, get_parameters


@st.cache_data(show_spinner=True)
def get_params_df(run_ids: List[str], experiment_id: str):
    """
    Fetches parameters for a list of run IDs from the MLflow store.

    Args:
        run_ids (List[str]): A list of run IDs.
        experiment_id (str): The experiment ID.

    Returns:
        pd.DataFrame: A DataFrame containing the parameters for the given run IDs.
    """
    run_params_list = []
    for run in run_ids:
        run_id = run["experiment_run_id"]
        run_params_request = MLFlowStoreIdentifier(
            experiment_id=experiment_id, run_id=run_id
        )
        run_params = get_parameters(run_params_request)
        run_params = {list(d.values())[0]: list(d.values())[1] for d in run_params}
        run_params["run_id"] = run_id

        run_params_list.append(run_params)

    if not run_params_list:
        return pd.DataFrame()

    return pd.DataFrame(run_params_list)


@st.cache_data(show_spinner=True)
def get_numeric_metrics_df(request: MLFlowStoreMetricRequest):
    """
    Retrieve numeric metrics from MLFlow store and return them as a DataFrame.

    Args:
        request (MLFlowStoreMetricRequest): The request object containing parameters to fetch metrics.

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


def get_df_from_json_list(json_list: List[Dict[str, Any]]) -> pd.DataFrame:
    """
    Converts a dictionary of lists to a pandas DataFrame.

    Args:
        json_data (Dict[str, Union[str, List[str]]]): A dictionary containing lists of data.

    Returns:
        pd.DataFrame: A DataFrame containing the data from the input dictionary.
    """
    keys_to_keep = ["run_id"]
    rows = []
    for json_dict in json_list:
        json_data = json_dict["value"]
        json_data["run_id"] = json_dict["experiment_run_id"]
        for key, value in json_data.items():
            if not isinstance(value, (list, dict)):
                keys_to_keep.append(key)
        new_json_data = {}
        for key in keys_to_keep:
            new_json_data[key] = json_data.get(key, None)
        rows.append(new_json_data)
    return pd.DataFrame(rows)


def get_df_from_json_dicts(json_dicts: Dict[str, List[Dict[str, Any]]]) -> pd.DataFrame:
    """
    Converts a dictionary of lists to a pandas DataFrame.

    Args:
        json_dict (Dict[str, List[Dict]]): A dictionary containing lists of data.

    Returns:
        pd.DataFrame: A DataFrame containing the data from the input dictionary.
    """
    json_dfs = {}
    for json_file, json_list in json_dicts.items():
        json_dfs[json_file] = get_df_from_json_list(json_list)
    json_df = reduce(
        lambda left, right: pd.merge(left, right, on="run_id", how="left"),
        [df for df in json_dfs.values()],
    )
    return json_df


def highlight_words(s, words):
    """
    Highlights words in a string with a background color.

    Args:
        s (str): The string to highlight.
        words (List[str]): A list of words to highlight in the string.

    Returns:
        str: The string with highlighted words.
    """
    for word in words:
        if word in s:
            s = s.replace(
                word,
                f'<span style="background-color: #f0f9eb; border-radius: 5px;">{word}</span>',
            )
    return s


def show_parameters_overview_component(
    params_df: pd.DataFrame,
):
    """
    Display the parameters overview component.

    Args:
        parameters_df (pd.DataFrame): DataFrame containing parameters.

    Returns:
        None
    """
    if not params_df.empty:
        st.write("### Parameters")
        params_container = st.container(border=True)
        with params_container:
            # Combined configuration across all runs
            if "run_id" in params_df.columns.to_list():
                params_df = params_df.drop(columns=["run_id"])
            column_names = params_df.columns
            most_common_config = params_df.value_counts().idxmax()
            counts = params_df.value_counts().max()
            st.write("**Most Common Configuration Parameters**")
            overview_param_cols = st.columns(len(column_names) + 1)
            for i, col in enumerate(overview_param_cols):
                if i == len(column_names):
                    col.write("Frequency")
                    col.caption(counts)
                else:
                    col.write(column_names[i].replace("_", " ").title())
                    col.caption(most_common_config[i])

            # Count the number of unique values for each parameter
            st.write("**Top Configuration Parameters**")
            param_cols = st.columns(len(column_names))
            for i, col in enumerate(param_cols):
                col.write(column_names[i].replace("_", " ").title())
                col.caption(
                    f"{params_df[column_names[i]].value_counts().idxmax()} "
                    f"({params_df[column_names[i]].value_counts().max()} times)"
                )


def show_detailed_logs_component(
    live_results_df: pd.DataFrame,
    metrics_dfs: List[pd.DataFrame],
):
    """
    Display detailed logs for live results and metrics.

    Args:
        live_results_df (pd.DataFrame): DataFrame containing live results.
        metrics_dfs (List[pd.DataFrame]): List of DataFrames containing metrics.

    Returns:
        None
    """
    if not live_results_df.empty:
        if metrics_dfs:
            metrics_dfs = [live_results_df] + metrics_dfs
            live_results_df = reduce(
                lambda left, right: pd.merge(left, right, on="run_id", how="left"),
                metrics_dfs,
            )
        if "response_id" in live_results_df.columns:
            live_results_df = live_results_df.drop(columns=["response_id"])
        if "run_id" in live_results_df.columns:
            live_results_df = live_results_df.drop(columns=["run_id"])

        if "feedback" in live_results_df.columns:
            live_results_df["feedback"] = live_results_df["feedback"].apply(
                lambda x: "👍" if x == 1 else "👎" if x == 0 else "🤷‍♂️"
            )
        with st.expander(":material/live_help: **Detailed Logs**", expanded=True):
            if not "timestamp" in live_results_df.columns:
                live_results_df = live_results_df.rename(
                    columns={
                        x: x.replace("_", " ").title() for x in live_results_df.columns
                    }
                )
                st.dataframe(live_results_df)
                return
            live_results_df["timestamp"] = pd.to_datetime(
                live_results_df["timestamp"], format="mixed", dayfirst=True
            )
            live_results_df.sort_values(
                by="timestamp", ascending=False, inplace=True, ignore_index=True
            )
            live_results_df = live_results_df.rename(
                columns={
                    x: x.replace("_", " ").title() for x in live_results_df.columns
                }
            )
            filtered_live_results_df = dataframe_explorer(
                df=live_results_df, case=False
            )
            st.dataframe(filtered_live_results_df)


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
    label: str,
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

        st.markdown(
            f"### {label}",
            help="Feedback received from users.",
        )

        thumbs_up_placeholder, thumbs_down_placeholder, no_feedback_placeholder = (
            st.columns(3)
        )

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


def show_feedback_kpi(
    metric_key: str,
    metrics_df: pd.DataFrame,
    kpi_placeholder: DeltaGenerator,
    label: str,
    tooltip: Optional[
        str
    ] = "Average pass rate of responses. Includes thumbs up and no feecback.",
):
    """
    Display feedback KPIs.

    Parameters:
    metric_key (str): The key to identify the metric in the DataFrame.
    metrics_df (pd.DataFrame): DataFrame containing feedback and timestamps.
    kpi_placeholder (DeltaGenerator): Streamlit placeholder for feedback KPI.
    label (str): The label for the feedback KPI.
    tooltip (str): The tooltip text for the feedback KPI.

    Returns:
    None
    """
    if metric_key in metrics_df:
        thumbs_down_count = metrics_df[metric_key].to_list().count(0)
        prev_thumbs_down_count = metrics_df[metric_key].to_list()[:-1].count(0)
        metric_value = (1 - (thumbs_down_count / len(metrics_df))) * 100
        metric_value = round(metric_value, 2)
        prev_metric_value = (1 - (prev_thumbs_down_count / (len(metrics_df) - 1))) * 100
        prev_metric_value = round(prev_metric_value, 2)
        delta_value = metric_value - prev_metric_value if len(metrics_df) > 1 else 0
        metric_value = f"{metric_value}%"
        kpi_placeholder.metric(
            label=label, help=tooltip, value=metric_value, delta=delta_value
        )


def show_numeric_metric_kpi(
    metric_key: str,
    metrics_df: pd.DataFrame,
    kpi_placeholder: DeltaGenerator,
    label: str,
    tooltip: Optional[str] = None,
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
    labels: Optional[List[str]] = None,
    update_timestamp: str = "",
    fig_placeholder: DeltaGenerator = None,
    tooltip: Optional[str] = None,
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
    fig_placeholder (DeltaGenerator): Streamlit placeholder for the pie chart.

    Returns:
    None
    """
    if metric_key in metrics_df:
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
        if fig_placeholder is None:
            st.markdown(f"### {title}", help=tooltip)
            st.plotly_chart(fig, key=f"{metric_key}_fig_{update_timestamp}")
            return
        fig_placeholder.markdown(f"### {title}", help=tooltip)
        fig_placeholder.plotly_chart(fig, key=f"{metric_key}_fig_{update_timestamp}")


def show_time_series_component(
    metric_key: str,
    metrics_df: pd.DataFrame,
    title: str,
    update_timestamp: str,
    frequency: str = "h",
    fig_placeholder: DeltaGenerator = None,
    tooltip: Optional[str] = None,
):
    """
    Displays a time series component in a Streamlit app.

    Parameters:
    metric_key (str): The key to identify the metric in the DataFrame.
    metrics_df (pd.DataFrame): The DataFrame containing the metrics data.
    update_timestamp (str): A timestamp string to ensure the chart is updated.
    title (str): The title of the time series plot.
    tooltip (str): The tooltip text for the time series plot title.
    frequency (str): The frequency to group the time series data.
    fig_placeholder (DeltaGenerator): Streamlit placeholder for the time series plot.

    Returns:
    None
    """
    if metric_key in metrics_df:
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
        if fig_placeholder is None:
            st.markdown(f"### {title}", help=tooltip)
            st.plotly_chart(fig, key=f"{metric_key}_fig_{update_timestamp}")
            return
        fig_placeholder.markdown(f"### {title}", help=tooltip)
        fig_placeholder.plotly_chart(fig, key=f"{metric_key}_fig_{update_timestamp}")


def keywords_in_dict(d: Dict):
    """
    Check if the dictionary contains keywords.

    Args:
        d (Dict): The dictionary to check for keywords.

    Returns:
        bool: True if the dictionary contains keywords, False otherwise.
    """
    if "query_keywords" in d or "response_keywords" in d:
        return True
    return False


def show_wordcloud_component(live_results_dict: List[Dict]):
    """
    Displays a word cloud component in Streamlit.

    Parameters:
    live_results_dict (List[Dict]): A list of dictionaries containing the live results.

    Returns:
    None
    """
    query_keywords = ""
    response_keywords = ""

    for d in live_results_dict:
        if "query_keywords" in d["value"]:
            query_keywords += d["value"]["query_keywords"]
        if "response_keywords" in d["value"]:
            response_keywords += d["value"]["response_keywords"]

    q_wc = WordCloud()
    q_fig = q_wc.generate(query_keywords)
    r_wc = WordCloud()
    r_fig = r_wc.generate(response_keywords)

    with st.expander(":material/label: **Word Cloud**", expanded=True):
        q_col, r_col = st.columns(2)
        if "query_keywords" != "":
            with q_col:
                st.markdown("### Query Keywords")
                st.image(q_fig.to_image(), use_container_width=True)
        if "response_keywords" != "":
            with r_col:
                st.markdown("### Response Keywords")
                st.image(r_fig.to_image(), use_container_width=True)
