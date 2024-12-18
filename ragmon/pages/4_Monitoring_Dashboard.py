# ###########################################################################
#
#  CLOUDERA APPLIED MACHINE LEARNING PROTOTYPE (AMP)
#  (C) Cloudera, Inc. 2021
#  All rights reserved.
#
#  Applicable Open Source License: Apache 2.0
#
#  NOTE: Cloudera open source products are modular software products
#  made up of hundreds of individual components, each of which was
#  individually copyrighted.  Each Cloudera open source product is a
#  collective work under U.S. Copyright Law. Your license to use the
#  collective work is as provided in your written agreement with
#  Cloudera.  Used apart from the collective work, this file is
#  licensed for your use pursuant to the open source license
#  identified above.
#
#  This code is provided to you pursuant a written agreement with
#  (i) Cloudera, Inc. or (ii) a third-party authorized to distribute
#  this code. If you do not have a written agreement with Cloudera nor
#  with an authorized and properly licensed third party, you do not
#  have any rights to access nor to use this code.
#
#  Absent a written agreement with Cloudera, Inc. (“Cloudera”) to the
#  contrary, A) CLOUDERA PROVIDES THIS CODE TO YOU WITHOUT WARRANTIES OF ANY
#  KIND; (B) CLOUDERA DISCLAIMS ANY AND ALL EXPRESS AND IMPLIED
#  WARRANTIES WITH RESPECT TO THIS CODE, INCLUDING BUT NOT LIMITED TO
#  IMPLIED WARRANTIES OF TITLE, NON-INFRINGEMENT, MERCHANTABILITY AND
#  FITNESS FOR A PARTICULAR PURPOSE; (C) CLOUDERA IS NOT LIABLE TO YOU,
#  AND WILL NOT DEFEND, INDEMNIFY, NOR HOLD YOU HARMLESS FOR ANY CLAIMS
#  ARISING FROM OR RELATED TO THE CODE; AND (D)WITH RESPECT TO YOUR EXERCISE
#  OF ANY RIGHTS GRANTED TO YOU FOR THE CODE, CLOUDERA IS NOT LIABLE FOR ANY
#  DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, PUNITIVE OR
#  CONSEQUENTIAL DAMAGES INCLUDING, BUT NOT LIMITED TO, DAMAGES
#  RELATED TO LOST REVENUE, LOST PROFITS, LOSS OF INCOME, LOSS OF
#  BUSINESS ADVANTAGE OR UNAVAILABILITY, OR LOSS OR CORRUPTION OF
#  DATA.
#
# ###########################################################################

from typing import List  # to simulate a real time data, time loop
import time
import os
from pathlib import Path
import logging
import json
import warnings
import numpy as np
import pandas as pd  # read csv, df manipulation
import plotly.graph_objects as go  # interactive charts
import streamlit as st  # 🎈 data web app development
import requests  # to pull data from metric store

from qdrant_client import QdrantClient
from data_types import MLFlowStoreRequest
from ragmon.utils.dashboard import (
    get_experiment_ids,
    get_runs,
    parse_live_results_table,
    get_numeric_metrics_df,
)

warnings.filterwarnings("ignore")

# get resources directory
file_path = Path(os.path.realpath(__file__))
st_app_dir = file_path.parents[1]
COLLECTIONS_JSON = os.path.join(st_app_dir, "collections.json")


def get_collections():
    """
    Retrieve a list of collections from the client.
    Returns:
        list: A list of collections retrieved from the client.
    """
    client = QdrantClient(url="http://localhost:6333")
    collections = client.get_collections().collections
    if len(collections) == 0:
        with open(COLLECTIONS_JSON, "w+") as f:
            collections = []
            json.dump(collections, f)
    else:
        with open(COLLECTIONS_JSON, "r+") as f:
            try:
                collections = json.load(f)
            except json.JSONDecodeError:
                collections = []
    client.close()
    return collections


title_col, refresh_col = st.columns([12, 1])
# dashboard title
with title_col:
    st.title(":material/monitoring: Monitoring Dashboard")

with refresh_col:
    if st.button(
        ":material/sync:",
        use_container_width=True,
        help="Refresh the dashboard for updated metrics",
    ):
        st.rerun()

# select experiment/data source
experiment_ids = get_experiment_ids()
collections = get_collections()

if not experiment_ids:
    st.write("No Data Sources or Entries Found")

if experiment_ids:
    experiment_ids.sort(key=lambda x: int(x))

    data_source_names = {
        exp_id: collection["name"]
        for exp_id, collection in zip(experiment_ids, collections)
    }

    selected_experiment = st.selectbox(
        "Select a Data Source :material/database:",
        options=experiment_ids,
        index=len(experiment_ids) - 1,
        format_func=lambda x: data_source_names[x],
    )

    # select run
    runs = get_runs(selected_experiment)

    if not runs:
        st.write("No Metrics Logged Yet")

    if runs:
        run_ids = [run["experiment_run_id"] for run in runs]

        mock_precision_scores = np.random.random(len(run_ids))
        mock_recall_scores = np.random.random(len(run_ids))

        # creating requests for metrics

        faithfulness_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["faithfulness_score"],
        )

        relevancy_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["relevance_score"],
        )

        context_relevancy_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["context_relevancy_score"],
        )

        input_lengths_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["input_length"],
        )

        output_lengths_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["output_length"],
        )

        maliciousness_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["maliciousness_score"],
        )

        toxicity_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["toxicity_score"],
        )

        comprehensiveness_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["comprehensiveness_score"],
        )

        precision_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["precision"],
        )

        recall_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["recall"],
        )

        feedback_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["feedback"],
        )

        table_request = MLFlowStoreRequest(
            experiment_id=str(selected_experiment),
            run_ids=run_ids,
            metric_names=["live_results.json"],
        )

        placeholder = st.empty()

        # near real-time / live feed simulation
        update_timestamp = time.strftime("%Y-%m-%d %H:%M:%S")

        # get live results
        live_results_df = parse_live_results_table(table_request)

        # get remaining metrics
        faithfulness_df = get_numeric_metrics_df(faithfulness_request)
        relevance_df = get_numeric_metrics_df(relevancy_request)
        context_relevancy_df = get_numeric_metrics_df(context_relevancy_request)
        input_lengths_df = get_numeric_metrics_df(input_lengths_request)
        output_lengths_df = get_numeric_metrics_df(output_lengths_request)
        maliciousness_df = get_numeric_metrics_df(maliciousness_request)
        toxicity_df = get_numeric_metrics_df(toxicity_request)
        comprehensiveness_df = get_numeric_metrics_df(comprehensiveness_request)
        precision_df = get_numeric_metrics_df(precision_request)
        recall_df = get_numeric_metrics_df(recall_request)
        feedback_df = get_numeric_metrics_df(feedback_request)

        with placeholder.container():

            kpi9, kpi10, kpi11, kpi12, kpi13 = st.columns([3, 3, 1, 1, 1])

            io_col, feedback_col = st.columns([3, 2])

            with io_col:
                # Graphs
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

                    output_lengths_df = live_results_df[["output_length", "timestamp"]]
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
                        st.plotly_chart(
                            fig, key=f"output_length_fig_{update_timestamp}"
                        )

            # Show thumbs up count, thumbs down count, no feedback count
            with feedback_col:
                thumbs_up_count = live_results_df["thumbs_up"].to_list().count(1)
                thumbs_down_count = live_results_df["thumbs_up"].to_list().count(0)
                no_feedback_count = live_results_df["thumbs_up"].isna().sum()

                kpi11.metric(
                    label="Thumbs Up :material/thumb_up:",
                    help="The number of thumbs up received.",
                    value=thumbs_up_count,
                )

                kpi12.metric(
                    label="Thumbs Down :material/thumb_down:",
                    help="The number of thumbs down received.",
                    value=thumbs_down_count,
                )

                kpi13.metric(
                    label="No Feedback :material/indeterminate_question_box:",
                    help="The number of no feedback received.",
                    value=no_feedback_count,
                )

                with st.expander(
                    "# :material/feedback: **Feedback Overview**", expanded=True
                ):
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

            # metric columns
            kpi1, kpi2, kpi3, kpi4, kpi5, kpi6, kpi7 = st.columns([1, 1, 1, 1, 1, 1, 2])

            if "faithfulness_score" in live_results_df.columns:
                avg_faithfulness = np.mean(live_results_df["faithfulness_score"])
                faithfulness_scores = live_results_df[
                    live_results_df["faithfulness_score"].notna()
                ]["faithfulness_score"].to_list()
                # fill in those three columns with respective metrics or KPIs
                kpi1.metric(
                    label="Faithfulness",
                    help="Faithfulness is the degree to which the model generates responses that are faithful to the input.",
                    value=round(avg_faithfulness, 2),
                    delta=round(
                        (
                            avg_faithfulness - np.mean(faithfulness_scores[:-1])
                            if len(faithfulness_scores) > 1
                            else 0
                        ),
                        2,
                    ),
                )

            if "relevance_score" in live_results_df.columns:
                avg_relevance = np.mean(live_results_df["relevance_score"])
                relevance_scores = live_results_df[
                    live_results_df["relevance_score"].notna()
                ]["relevance_score"].to_list()
                kpi2.metric(
                    label="Relevance",
                    help="Relevance is the degree to which the model generates responses that are relevant to the input.",
                    value=round(avg_relevance, 2),
                    delta=round(
                        (
                            avg_relevance - np.mean(relevance_scores[:-1])
                            if len(relevance_scores) > 1
                            else 0
                        ),
                        2,
                    ),
                )

            if "context_relevancy_score" in live_results_df.columns:
                avg_context_relevance = np.mean(
                    live_results_df["context_relevancy_score"]
                )
                context_relevancy_scores = live_results_df[
                    live_results_df["context_relevancy_score"].notna()
                ]["context_relevancy_score"].to_list()
                kpi3.metric(
                    label="Context Relevance",
                    help="Context Relevance is the degree to which contexts retrieved are contextually relevant.",
                    value=round(avg_context_relevance, 2),
                    delta=round(
                        (
                            avg_context_relevance
                            - np.mean(context_relevancy_scores[:-1])
                            if len(context_relevancy_scores) > 1
                            else 0
                        ),
                        2,
                    ),
                )

            if "maliciousness_score" in live_results_df.columns:
                avg_maliciousness = np.mean(live_results_df["maliciousness_score"])
                maliciousness_scores = live_results_df[
                    live_results_df["maliciousness_score"].notna()
                ]["maliciousness_score"].to_list()
                kpi4.metric(
                    label="Maliciousness",
                    help="The degree to which the model generates responses that are malicious or harmful.",
                    value=round(avg_maliciousness, 2),
                    delta=round(
                        (
                            avg_maliciousness - np.mean(maliciousness_scores[:-1])
                            if len(maliciousness_scores) > 1
                            else 0
                        ),
                        2,
                    ),
                    delta_color="inverse",
                )

            if "toxicity_score" in live_results_df.columns:
                avg_toxicity = np.mean(live_results_df["toxicity_score"])
                toxicity_scores = live_results_df[
                    live_results_df["toxicity_score"].notna()
                ]["toxicity_score"].to_list()
                kpi5.metric(
                    label="Toxicity",
                    help="The degree to which the model generates responses that are toxic.",
                    value=round(avg_toxicity, 2),
                    delta=round(
                        (
                            avg_toxicity - np.mean(toxicity_scores[:-1])
                            if len(toxicity_scores) > 1
                            else 0
                        ),
                        2,
                    ),
                    delta_color="inverse",
                )

            if "comprehensiveness_score" in live_results_df.columns:
                avg_comprehensiveness = np.mean(
                    live_results_df["comprehensiveness_score"]
                )
                comprehensiveness_scores = live_results_df[
                    live_results_df["comprehensiveness_score"].notna()
                ]["comprehensiveness_score"].to_list()
                kpi6.metric(
                    label="Comprehensiveness",
                    help="The degree to which the model generates responses that are comprehensive.",
                    value=round(avg_comprehensiveness, 2),
                    delta=round(
                        (
                            avg_comprehensiveness
                            - np.mean(comprehensiveness_scores[:-1])
                            if len(comprehensiveness_scores) > 1
                            else 0
                        ),
                        2,
                    ),
                )

            with kpi7.container():
                metric_col1, metric_col2 = st.columns(2)
                precision_scores = mock_precision_scores
                avg_precision = np.mean(precision_scores)
                metric_col1.metric(
                    label="Precision",
                    help="The precision of contexts retrieved.",
                    value=round(avg_precision, 2),
                )

                recall_scores = mock_recall_scores
                avg_recall = np.mean(recall_scores)
                metric_col2.metric(
                    label="Recall",
                    help="The recall of contexts retrieved.",
                    value=round(avg_recall, 2),
                )
                st.caption("Coming Soon! :material/sunny:")

            # Graphs
            with st.expander(
                ":material/analytics: **Metrics Overview**", expanded=True
            ):
                fig_col1, fig_col2, fig_col3 = st.columns(3)
                with fig_col1:
                    st.markdown(
                        "### Faithfulness",
                        help="Faithfulness of the answer received.",
                    )
                    fig = go.Figure(
                        data=go.Pie(
                            labels=["Faithful", "Not Faithful", "No Records"],
                            values=[
                                live_results_df["faithfulness_score"]
                                .to_list()
                                .count(1),
                                live_results_df["faithfulness_score"]
                                .to_list()
                                .count(0),
                                live_results_df["faithfulness_score"].isna().sum(),
                            ],
                            hole=0.5,
                            hovertemplate="%{label}: <b>%{value}</b><extra></extra>",
                        )
                    )
                    st.plotly_chart(fig, key=f"faithfulness_fig_{update_timestamp}")

                with fig_col2:
                    st.markdown(
                        "### Relevance", help="Relevance of the answer received."
                    )
                    fig = go.Figure(
                        data=go.Pie(
                            labels=["Relevant", "Not Relevant", "No Records"],
                            values=[
                                live_results_df["relevance_score"].to_list().count(1),
                                live_results_df["relevance_score"].to_list().count(0),
                                live_results_df["relevance_score"].isna().sum(),
                            ],
                            hole=0.5,
                            hovertemplate="%{label}: <b>%{value}</b><extra></extra>",
                        )
                    )
                    st.plotly_chart(fig, key=f"relevance_fig_{update_timestamp}")

                with fig_col3:
                    context_relevancy_df = live_results_df[
                        live_results_df["context_relevancy_score"].notnull()
                    ][["context_relevancy_score", "timestamp"]]
                    agg_context_relevancy_df = context_relevancy_df.groupby(
                        pd.Grouper(key="timestamp", freq="h")  # group by hour
                    )["context_relevancy_score"].agg(["mean", "max", "min"])
                    st.markdown(
                        "### Context Relevance Score",
                        help="Relevance of the contexts received.",
                    )
                    fig = go.Figure(
                        data=go.Scatter(
                            x=agg_context_relevancy_df.index,
                            y=agg_context_relevancy_df["mean"],
                            mode="markers",
                            marker=dict(size=5),
                            fill="tozeroy",
                            customdata=agg_context_relevancy_df[["max", "min"]],
                            hovertemplate="Mean: <b>%{y:.2f}</b> Max: <b>%{customdata[0]:.2f}"
                            "</b><br>Min: <b>%{customdata[1]:.2f}</b><br>Date: %{x|%b %d, %Y}"
                            "<br>Time: %{x|%H:%M}<extra></extra>",
                        )
                    )
                    fig.update_layout(
                        xaxis_title="Date",
                        yaxis_title="Mean Context Relevance Score (0-1)",
                        yaxis=dict(range=[0, 1]),
                        xaxis={
                            "tickformat": "%b %d, %Y",
                            "tickmode": "array",
                        },
                    )
                    st.plotly_chart(
                        fig, key=f"context_relevance_fig_{update_timestamp}"
                    )

                fig_col4, fig_col5, fig_col6 = st.columns(3)

                with fig_col4:
                    maliciousness_df = live_results_df[
                        live_results_df["maliciousness_score"].notnull()
                    ][["maliciousness_score", "timestamp"]]
                    agg_maliciousness_df = maliciousness_df.groupby(
                        pd.Grouper(key="timestamp", freq="h")  # group by hour
                    )["maliciousness_score"].agg(["mean", "max", "min"])
                    st.markdown(
                        "### Maliciousness",
                        help="Maliciousness of the answer received.",
                    )
                    fig = go.Figure(
                        data=go.Scatter(
                            x=agg_maliciousness_df.index,
                            y=agg_maliciousness_df["mean"],
                            mode="markers",
                            marker=dict(size=5),
                            fill="tozeroy",
                            customdata=agg_maliciousness_df[["max", "min"]],
                            hovertemplate="Mean: <b>%{y:.2f}</b> Max: <b>%{customdata[0]:.2f}"
                            "</b><br>Min: <b>%{customdata[1]:.2f}</b><br>Date: %{x|%b %d, %Y}"
                            "<br>Time: %{x|%H:%M}<extra></extra>",
                        )
                    )
                    fig.update_layout(
                        xaxis_title="Date",
                        yaxis_title="Mean Maliciousness Score (0-1)",
                        yaxis=dict(range=[0, 1]),
                        xaxis={
                            "tickformat": "%b %d, %Y",
                            "tickmode": "array",
                        },
                    )
                    st.plotly_chart(fig, key=f"maliciousness_fig_{update_timestamp}")

                with fig_col5:
                    toxicity_df = live_results_df[
                        live_results_df["toxicity_score"].notnull()
                    ][["toxicity_score", "timestamp"]]
                    agg_toxicity_df = toxicity_df.groupby(
                        pd.Grouper(key="timestamp", freq="h")  # group by hour
                    )["toxicity_score"].agg(["mean", "max", "min"])
                    st.markdown("### Toxicity", help="Toxicity of the answer received.")
                    fig = go.Figure(
                        data=go.Scatter(
                            x=agg_toxicity_df.index,
                            y=agg_toxicity_df["mean"],
                            mode="markers",
                            marker=dict(size=5),
                            fill="tozeroy",
                            customdata=agg_toxicity_df[["max", "min"]],
                            hovertemplate="Mean: <b>%{y:.2f}</b> Max: <b>%{customdata[0]:.2f}"
                            "</b><br>Min: <b>%{customdata[1]:.2f}</b><br>Date:%{x|%b %d, %Y}"
                            "<br>Time: %{x|%H:%M}<extra></extra>",
                        )
                    )
                    fig.update_layout(
                        xaxis_title="Date",
                        yaxis_title="Mean Toxicity Score (0-1)",
                        yaxis=dict(range=[0, 1]),
                        xaxis={
                            "tickformat": "%b %d, %Y",
                            "tickmode": "array",
                        },
                    )
                    st.plotly_chart(fig, key=f"toxicity_fig_{update_timestamp}")

                with fig_col6:
                    comprehensiveness_df = live_results_df[
                        live_results_df["comprehensiveness_score"].notnull()
                    ]
                    agg_comprehensiveness_df = comprehensiveness_df.groupby(
                        pd.Grouper(key="timestamp", freq="h")  # group by hour
                    )["comprehensiveness_score"].agg(["mean", "min", "max"])
                    st.markdown(
                        "### Comprehensiveness",
                        help="Comprehensiveness of the answer.",
                    )
                    fig = go.Figure(
                        data=go.Scatter(
                            x=agg_comprehensiveness_df.index,
                            y=agg_comprehensiveness_df["mean"],
                            mode="markers",
                            marker=dict(size=5),
                            fill="tozeroy",
                            customdata=agg_comprehensiveness_df[["max", "min"]],
                            hovertemplate="Mean: <b>%{y:.2f}</b> Max: <b>%{customdata[0]:.2f}"
                            "</b><br>Min: <b>%{customdata[1]:.2f}</b><br>Date:%{x|%b %d, %Y}"
                            "<br>Time: %{x|%H:%M}<extra></extra>",
                        )
                    )
                    fig.update_layout(
                        xaxis_title="Date",
                        yaxis_title="Mean Comprehensiveness Score (0-1)",
                        yaxis=dict(range=[0, 1]),
                        xaxis={
                            "tickformat": "%b %d, %Y",
                            "tickmode": "array",
                        },
                    )
                    st.plotly_chart(
                        fig, key=f"comprehensiveness_fig_{update_timestamp}"
                    )

            if not live_results_df.empty:
                live_results_df = live_results_df.drop(
                    columns=["response_id", "run_id"]
                )
                live_results_df["thumbs_up"] = live_results_df["thumbs_up"].apply(
                    lambda x: "👍" if x == 1 else "👎" if x == 0 else "🤷‍♂️"
                )
                live_results_df = live_results_df.rename(
                    {
                        # "feedback_str": "user_feedback",
                        "thumbs_up": "feedback"
                    },
                    axis=1,
                )
                st.write("### Detailed Logs")
                st.dataframe(
                    live_results_df.sort_values(by="timestamp", ascending=False)
                )
