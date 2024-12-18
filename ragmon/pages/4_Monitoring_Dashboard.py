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
    show_i_o_component,
    show_feedback_component,
    show_numeric_metric_kpi,
    show_live_df_component,
    show_pie_chart_component,
    show_time_series_component,
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
                show_i_o_component(
                    input_lengths_df=input_lengths_df,
                    output_lengths_df=output_lengths_df,
                    input_kpi_placeholder=kpi9,
                    output_kpi_placeholder=kpi10,
                    update_timestamp=update_timestamp,
                )

            # Show thumbs up count, thumbs down count, no feedback count
            with feedback_col:
                show_feedback_component(
                    feedback_df=feedback_df,
                    thumbs_down_placeholder=kpi11,
                    thumbs_up_placeholder=kpi12,
                    no_feedback_placeholder=kpi13,
                    update_timestamp=update_timestamp,
                )

            # metric columns
            kpi1, kpi2, kpi3, kpi4, kpi5, kpi6, kpi7 = st.columns([1, 1, 1, 1, 1, 1, 2])

            show_numeric_metric_kpi(
                metric_key="faithfulness_score",
                metrics_df=faithfulness_df,
                kpi_placeholder=kpi1,
                label="Faithfulness",
                tooltip="Faithfulness is the degree to which the model"
                " generates responses that are faithful to the input.",
            )

            show_numeric_metric_kpi(
                metric_key="relevance_score",
                metrics_df=relevance_df,
                kpi_placeholder=kpi2,
                label="Relevance",
                tooltip="Relevance is the degree to which the model generates"
                " responses that are relevant to the input.",
            )

            show_numeric_metric_kpi(
                metric_key="context_relevancy_score",
                metrics_df=context_relevancy_df,
                kpi_placeholder=kpi3,
                label="Context Relevance",
                tooltip="Context Relevance is the degree to which contexts"
                " retrieved are contextually relevant.",
            )

            show_numeric_metric_kpi(
                metric_key="maliciousness_score",
                metrics_df=maliciousness_df,
                kpi_placeholder=kpi4,
                label="Maliciousness",
                tooltip="The degree to which the model generates responses"
                " that are malicious or harmful.",
            )

            show_numeric_metric_kpi(
                metric_key="toxicity_score",
                metrics_df=toxicity_df,
                kpi_placeholder=kpi5,
                label="Toxicity",
                tooltip="The degree to which the model generates responses"
                " that are toxic.",
            )

            show_numeric_metric_kpi(
                metric_key="comprehensiveness_score",
                metrics_df=comprehensiveness_df,
                kpi_placeholder=kpi6,
                label="Comprehensiveness",
                tooltip="The degree to which the model generates responses"
                " that are comprehensive.",
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
                    show_pie_chart_component(
                        metric_key="faithfulness_score",
                        metrics_df=faithfulness_df,
                        title="Faithfulness",
                        tooltip="Faithfulness of the answer received.",
                        labels=["Faithful", "Not Faithful", "No Records"],
                        update_timestamp=update_timestamp,
                    )

                with fig_col2:
                    show_pie_chart_component(
                        metric_key="relevance_score",
                        metrics_df=relevance_df,
                        title="Relevance",
                        tooltip="Relevance of the answer received.",
                        labels=["Relevant", "Not Relevant", "No Records"],
                        update_timestamp=update_timestamp,
                    )

                with fig_col3:
                    show_time_series_component(
                        metric_key="context_relevancy_score",
                        metrics_df=context_relevancy_df,
                        title="Context Relevance",
                        tooltip="Relevance of the contexts received.",
                        update_timestamp=update_timestamp,
                        frequency="h",
                    )

                fig_col4, fig_col5, fig_col6 = st.columns(3)

                with fig_col4:
                    show_time_series_component(
                        metric_key="maliciousness_score",
                        metrics_df=maliciousness_df,
                        title="Maliciousness",
                        tooltip="Maliciousness of the answer received.",
                        update_timestamp=update_timestamp,
                        frequency="h",
                    )

                with fig_col5:
                    show_time_series_component(
                        metric_key="toxicity_score",
                        metrics_df=toxicity_df,
                        title="Toxicity",
                        tooltip="Toxicity of the answer received.",
                        update_timestamp=update_timestamp,
                        frequency="h",
                    )
                with fig_col6:
                    show_time_series_component(
                        metric_key="comprehensiveness_score",
                        metrics_df=comprehensiveness_df,
                        title="Comprehensiveness",
                        tooltip="Comprehensiveness of the answer received.",
                        update_timestamp=update_timestamp,
                        frequency="h",
                    )

            show_live_df_component(live_results_df)
