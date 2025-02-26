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

from functools import reduce
import time
import os
from pathlib import Path
import json
import warnings
import numpy as np
import pandas as pd  # read csv, df manipulation
import plotly.graph_objects as go  # interactive charts
import streamlit as st  # 🎈 data web app development

from data_types import (
    MLFlowExperimentRequest,
    MLFlowStoreMetricRequest,
)
from utils.dashboard import (
    get_experiments,
    get_runs,
    get_metric_names,
    get_params_df,
    get_numeric_metrics_df,
    get_json,
    get_df_from_json_dicts,
    show_parameters_overview_component,
    show_feedback_component,
    show_feedback_kpi,
    show_numeric_metric_kpi,
    show_pie_chart_component,
    show_time_series_component,
    keywords_in_dict,
    show_wordcloud_component,
    show_detailed_logs_component,
)
from utils.custom_evals import show_custom_evaluators_component

warnings.filterwarnings("ignore")

title_col, refresh_col = st.columns([12, 1])
# dashboard title
with title_col:
    st.title(":material/monitoring: RAG Studio")

with refresh_col:
    if st.button(
        ":material/sync:",
        use_container_width=True,
        help="Refresh the dashboard for updated metrics",
    ):
        st.rerun()

# select experiment/data source
experiments = get_experiments()

if not experiments:
    st.write("No Data Sources or Entries Found")

if experiments:
    selected_experiment = st.selectbox(
        "Select a Data Source :material/database:",
        options=experiments,
        index=len(experiments) - 1,
        format_func=lambda x: x["name"],
    )

    selected_experiment_id = selected_experiment["experiment_id"]
    selected_experiment_request = MLFlowExperimentRequest(
        experiment_id=str(selected_experiment_id)
    )
    # create requests for metric names, get metric names and sort it
    metric_names = get_metric_names(selected_experiment_request)
    metric_names = sorted(metric_names)

    numeric_metrics = [x for x in metric_names if not x.endswith(".json")]
    json_files = [x for x in metric_names if x.endswith(".json")]

    dashboard_tab, settings_tab = st.tabs(
        [":material/monitoring: Dashboard", ":material/settings: Settings"]
    )

    with settings_tab:
        with st.expander(":material/table_chart_view: Graph Settings", expanded=True):
            metrics_to_show = st.multiselect(
                "Select Metrics to Show",
                options=numeric_metrics,
                default=numeric_metrics,
            )
            st.write("##### Select Graph Type for Metrics")
            col_1, col_2 = st.columns([1, 1])
            graph_settings_dict = {}
            for i, metric_name in enumerate(metrics_to_show):
                if i % 2 == 0:
                    col_to_use = col_1
                else:
                    col_to_use = col_2
                if not "feedback" in metric_name.lower():
                    graph_settings_dict[metric_name] = col_to_use.radio(
                        f"{metric_name.replace('_', ' ').title()}",
                        [
                            ":material/timeline: Line Chart",
                            ":material/pie_chart: Pie Chart",
                        ],
                        index=(
                            1
                            if "faithfulness" in metric_name.lower()
                            or "relevance" in metric_name.lower()
                            else 0
                        ),
                        horizontal=True,
                    )
            st.write("##### Additional Settings")
            checkbox_col_1, checkbox_col_2 = st.columns([1, 1])
            wc_checkbox = checkbox_col_1.checkbox(
                "Show Wordcloud for Keywords",
                help="Show wordcloud for keywords in the selected json file",
                value=True,
            )
            logs_checkbox = checkbox_col_2.checkbox(
                "Show Detailed Logs",
                help="Show detailed logs for the selected experiment",
                value=True,
            )

    # get all runs for the selected experiment
    runs = get_runs(selected_experiment_request)

    with dashboard_tab:

        if not runs:
            st.write("No Metrics Logged Yet")

        if runs:
            run_ids = [run["experiment_run_id"] for run in runs]

            mock_precision_scores = np.random.random(len(run_ids))
            mock_recall_scores = np.random.random(len(run_ids))

            # get parameters and construct a dataframe
            params_df = get_params_df(
                run_ids=runs, experiment_id=selected_experiment_id
            )

            # show parameters overview
            show_parameters_overview_component(params_df)

            # create requests for metrics
            numeric_metrics_requests = {}

            for metric_name in numeric_metrics:
                metric_request = MLFlowStoreMetricRequest(
                    experiment_id=str(selected_experiment_id),
                    run_ids=run_ids,
                    metric_names=[metric_name],
                )
                numeric_metrics_requests[metric_name] = metric_request

            st.write("### Metrics")
            placeholder = st.container()

            # near real-time / live feed simulation
            update_timestamp = time.strftime("%Y-%m-%d %H:%M:%S")

            # get metrics responses
            metric_dfs = {}
            for metric_name, metric_request in numeric_metrics_requests.items():
                metric_dfs[metric_name] = get_numeric_metrics_df(metric_request)

            with placeholder:
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
                        if "feedback" in metric_name.lower():
                            metric_fig = metric_fig_rows[i // 3][i % 3]
                            with metric_fig:
                                feedback_df = metric_df
                                show_feedback_component(
                                    feedback_df=feedback_df,
                                    label=metric_name.replace("_", " ").title(),
                                    update_timestamp=update_timestamp,
                                )
                        else:
                            metric_fig = metric_fig_rows[i // 3][i % 3]
                            if (
                                graph_settings_dict.get(metric_name, None)
                                == ":material/pie_chart: Pie Chart"
                            ):
                                if "faithfulness" in metric_name.lower():
                                    labels = ["Faithful", "Not Faithful"]
                                if "relevance" in metric_name.lower():
                                    labels = ["Relevant", "Not Relevant"]
                                else:
                                    labels = None
                                show_pie_chart_component(
                                    metric_key=metric_name,
                                    metrics_df=metric_df,
                                    title=f"{metric_name.replace('_', ' ').title()}",
                                    labels=labels,
                                    update_timestamp=update_timestamp,
                                    fig_placeholder=metric_fig,
                                )
                            else:
                                show_time_series_component(
                                    metric_key=metric_name,
                                    metrics_df=metric_df,
                                    title=f"{metric_name.replace('_', ' ').title()}",
                                    update_timestamp=update_timestamp,
                                    frequency="h",
                                    fig_placeholder=metric_fig,
                                )

                json_dicts = {}

                # build dataframes from json files
                if json_files:
                    # Get logged json files
                    for json_file in json_files:
                        json_file_request = MLFlowStoreMetricRequest(
                            experiment_id=str(selected_experiment_id),
                            run_ids=run_ids,
                            metric_names=[json_file],
                        )
                        json_dicts[json_file] = get_json(json_file_request)

                # Find json file which contains the keywords
                if json_dicts:
                    if wc_checkbox:
                        keywords_file = None
                        for json_file, json_list in json_dicts.items():
                            for d in json_list:
                                if keywords_in_dict(d["value"]):
                                    keywords_file = json_file
                                    break
                            if keywords_file:
                                break

                        # Show keywords wordcloud
                        if keywords_file:
                            dict_w_keyword = json_dicts.get(keywords_file, None)
                            show_wordcloud_component(
                                live_results_dict=dict_w_keyword,
                            )

                if logs_checkbox:
                    if json_dicts:
                        # build dataframes from json files
                        json_df = get_df_from_json_dicts(json_dicts)

                        # check common columns in both dataframes except
                        common_columns = list(
                            set(json_df.drop(columns=["run_id"]).columns).intersection(
                                set(params_df.drop(columns=["run_id"]).columns)
                            )
                        )
                        if common_columns:
                            params_df = params_df.drop(columns=common_columns)

                        # merge json and params dataframes
                        params_df = pd.merge(
                            json_df, params_df, on="run_id", how="left"
                        )
                    metrics_dfs = [
                        df.drop(columns=["timestamp"])
                        for _, df in metric_dfs.items()
                        if not df.empty
                    ]
                    show_detailed_logs_component(params_df, metrics_dfs=metrics_dfs)

    with settings_tab:
        show_custom_evaluators_component()
