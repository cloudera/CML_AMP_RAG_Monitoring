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

import time
import os
from pathlib import Path
import json
import warnings
import numpy as np
import pandas as pd  # read csv, df manipulation
import plotly.graph_objects as go  # interactive charts
import streamlit as st  # 🎈 data web app development

from qdrant_client import QdrantClient
from data_types import (
    MLFlowExperimentRequest,
    MLFlowStoreMetricRequest,
    MLFlowStoreIdentifier,
)
from utils import get_collections
from utils.dashboard import (
    get_custom_evaluators,
    get_experiment_ids,
    get_runs,
    get_metric_names,
    parse_live_results_table,
    get_numeric_metrics_df,
    show_i_o_component,
    show_feedback_component,
    show_feedback_kpi,
    show_numeric_metric_kpi,
    show_live_df_component,
    show_pie_chart_component,
    show_time_series_component,
    show_wordcloud_component,
)

warnings.filterwarnings("ignore")

# get resources directory
file_path = Path(os.path.realpath(__file__))
st_app_dir = file_path.parents[1]
data_dir = os.path.join(st_app_dir, "data")
cols_dir = os.path.join(data_dir, "collections")
COLLECTIONS_JSON = os.path.join(cols_dir, "collections.json")
CUSTOM_EVAL_DIR = os.path.join(data_dir, "custom_evaluators")


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
collections = get_collections(COLLECTIONS_JSON=COLLECTIONS_JSON)
custom_evals = get_custom_evaluators(custom_evals_dir=CUSTOM_EVAL_DIR)

if not experiment_ids:
    st.write("No Data Sources or Entries Found")

if experiment_ids:
    data_source_names = {
        collection["mlflow_exp_id"]: collection["name"]
        for collection in collections
        if collection["mlflow_exp_id"] in experiment_ids
    }

    selected_experiment = st.selectbox(
        "Select a Data Source :material/database:",
        options=experiment_ids,
        index=len(experiment_ids) - 1,
        format_func=lambda x: data_source_names[x],
    )

    selected_experiment_request = MLFlowExperimentRequest(
        experiment_id=str(selected_experiment)
    )

    # select run
    runs = get_runs(selected_experiment_request)

    if not runs:
        st.write("No Metrics Logged Yet")

    if runs:
        run_ids = [run["experiment_run_id"] for run in runs]

        mock_precision_scores = np.random.random(len(run_ids))
        mock_recall_scores = np.random.random(len(run_ids))

        # create requests for metric names, get metric names and sort it
        metric_names = get_metric_names(selected_experiment_request)
        metric_names = sorted(metric_names)

        numeric_metrics = [x for x in metric_names if not x.endswith(".json")]
        non_numeric_metrics = [x for x in metric_names if x.endswith(".json")]

        # create requests for metrics
        numeric_metrics_requests = {}

        for metric_name in numeric_metrics:
            metric_request = MLFlowStoreMetricRequest(
                experiment_id=str(selected_experiment),
                run_ids=run_ids,
                metric_names=[metric_name],
            )
            numeric_metrics_requests[metric_name] = metric_request

        placeholder = st.empty()

        # near real-time / live feed simulation
        update_timestamp = time.strftime("%Y-%m-%d %H:%M:%S")

        # get metrics responses
        metric_dfs = {}
        for metric_name, metric_request in numeric_metrics_requests.items():
            metric_dfs[metric_name] = get_numeric_metrics_df(metric_request)

        with placeholder.container():
            # Non empty metrics
            non_empty_metrics = [
                metric_name
                for metric_name, metric_df in metric_dfs.items()
                if not metric_df.empty
            ]
            if non_empty_metrics:
                metric_rows = [
                    st.columns([1, 1, 1, 1, 1, 1])
                    for _ in range(
                        len(non_empty_metrics) // 6 + 1
                        if len(non_empty_metrics) % 6 != 0
                        else len(non_empty_metrics) // 6
                    )
                ]
                with st.expander(
                    ":material/analytics: **Metrics Overview**", expanded=True
                ):
                    metric_fig_rows = [
                        st.columns([1, 1, 1], border=False)
                        for _ in range(
                            len(non_empty_metrics) // 3 + 1
                            if len(non_empty_metrics) % 3 != 0
                            else len(non_empty_metrics) // 3
                        )
                    ]
                for i, metric_name in enumerate(non_empty_metrics):
                    metric_df = metric_dfs[metric_name]
                    metric_kpi = metric_rows[i // 6][i % 6]
                    if not "feedback" in metric_name.lower():
                        show_numeric_metric_kpi(
                            metric_key=metric_name,
                            metrics_df=metric_df,
                            kpi_placeholder=metric_kpi,
                            label=metric_name.replace("_", " ").title(),
                            tooltip=f"Average {metric_name.replace('_', ' ').title()}",
                        )
                    else:
                        show_feedback_kpi(
                            metric_key=metric_name,
                            metrics_df=metric_df,
                            kpi_placeholder=metric_kpi,
                            label=metric_name.replace("_", " ").title(),
                        )
                    if "faithfulness" in metric_name.lower():
                        metric_fig = metric_fig_rows[i // 3][i % 3]
                        show_pie_chart_component(
                            metric_key=metric_name,
                            metrics_df=metric_df,
                            title=f"{metric_name.replace('_', ' ').title()}",
                            labels=["Faithful", "Not Faithful"],
                            update_timestamp=update_timestamp,
                            fig_placeholder=metric_fig,
                        )
                    elif "relevance" in metric_name.lower():
                        metric_fig = metric_fig_rows[i // 3][i % 3]
                        show_pie_chart_component(
                            metric_key=metric_name,
                            metrics_df=metric_df,
                            title=f"{metric_name.replace('_', ' ').title()}",
                            labels=["Relevant", "Not Relevant"],
                            update_timestamp=update_timestamp,
                            fig_placeholder=metric_fig,
                        )
                    elif "feedback" in metric_name.lower():
                        metric_fig = metric_fig_rows[i // 3][i % 3]
                        with metric_fig:
                            kpi_cols = st.columns([1, 1, 1])
                            feedback_df = metric_df
                            show_feedback_component(
                                feedback_df=feedback_df,
                                thumbs_up_placeholder=kpi_cols[0],
                                thumbs_down_placeholder=kpi_cols[1],
                                no_feedback_placeholder=kpi_cols[2],
                                update_timestamp=update_timestamp,
                            )
                    else:
                        metric_fig = metric_fig_rows[i // 3][i % 3]
                        show_time_series_component(
                            metric_key=metric_name,
                            metrics_df=metric_df,
                            title=f"{metric_name.replace('_', ' ').title()}",
                            update_timestamp=update_timestamp,
                            frequency="h",
                            fig_placeholder=metric_fig,
                        )

            # TODO: reimplement of detailed logs
            # Show keywords wordcloud
            # show_wordcloud_component(
            #     df=live_results_df,
            # )

            # Live Results
            # metrics_dfs = [
            #     faithfulness_df.drop(columns=["timestamp"]),
            #     relevance_df.drop(columns=["timestamp"]),
            #     context_relevancy_df.drop(columns=["timestamp"]),
            #     maliciousness_df.drop(columns=["timestamp"]),
            #     toxicity_df.drop(columns=["timestamp"]),
            #     comprehensiveness_df.drop(columns=["timestamp"]),
            #     feedback_df.drop(columns=["timestamp"]),
            # ]

            # # append custom metrics to metrics_dfs
            # for _, custom_metric_df in custom_metrics_dfs.items():
            #     metrics_dfs.append(custom_metric_df.drop(columns=["timestamp"]))

            # show_live_df_component(live_results_df, metrics_dfs=metrics_dfs)
