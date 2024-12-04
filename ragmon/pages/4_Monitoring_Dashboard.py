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

warnings.filterwarnings("ignore")

# get resources directory
file_path = Path(os.path.realpath(__file__))
st_app_dir = file_path.parents[1]
COLLECTIONS_JSON = os.path.join(st_app_dir, "collections.json")

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


def get_runs(experiment_id: str):
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


def parse_live_results(results):
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

# top-level filters
# select experiment
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

        # creating a single-element container

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

        thumbs_up_request = MLFlowStoreRequest(
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

        # get live results logs
        live_results_response = get_metrics(table_request)

        if live_results_response == []:
            st.write("No Metrics Logged Yet")
        else:
            live_results_df = parse_live_results(live_results_response)

            # get remaining metrics
            faithfulness_response = get_metrics(faithfulness_request)
            relevance_response = get_metrics(relevancy_request)
            context_relevancy_response = get_metrics(context_relevancy_request)
            input_lengths_response = get_metrics(input_lengths_request)
            output_lengths_response = get_metrics(output_lengths_request)
            maliciousness_response = get_metrics(maliciousness_request)
            toxicity_response = get_metrics(toxicity_request)
            comprehensiveness_response = get_metrics(comprehensiveness_request)
            precision_response = get_metrics(precision_request)
            recall_response = get_metrics(recall_request)
            thumbs_up_response = get_metrics(thumbs_up_request)

            # initilize empty lists for all the scores
            faithfulness_scores = []
            relevance_scores = []
            context_relevancy_scores = []
            maliciousness_scores = []
            toxicity_scores = []
            comprehensiveness_scores = []
            thumbs_up_scores = []
            retrieval_precision_scores = []
            retrieval_recall_scores = []

            if thumbs_up_response != []:
                thumbs_up_response_ids = [
                    x["experiment_run_id"] for x in thumbs_up_response
                ]
                thumbs_up_scores = [
                    x["value"]["numericValue"] if "numericValue" in x["value"] else 0
                    for x in thumbs_up_response
                ]
                thumbs_up_df = pd.DataFrame(
                    {
                        "run_id": thumbs_up_response_ids,
                        "thumbs_up": thumbs_up_scores,
                    }
                )
            else:
                thumbs_up_df = pd.DataFrame(
                    {
                        "run_id": run_ids,
                        "thumbs_up": None,
                    }
                )
            live_results_df = live_results_df.merge(
                thumbs_up_df,
                on="run_id",
                how="left",
            )

            if faithfulness_response != []:
                faithfulness_response_ids = [
                    x["experiment_run_id"] for x in faithfulness_response
                ]
                faithfulness_scores = [
                    x["value"]["numericValue"] if "numericValue" in x["value"] else 0
                    for x in faithfulness_response
                ]
                faithfulness_df = pd.DataFrame(
                    {
                        "run_id": faithfulness_response_ids,
                        "faithfulness_score": faithfulness_scores,
                    }
                )
            else:
                faithfulness_df = pd.DataFrame(
                    {
                        "run_id": run_ids,
                        "faithfulness_score": pd.NA,
                    }
                )
            live_results_df = live_results_df.merge(
                faithfulness_df,
                on="run_id",
                how="left",
            )

            if relevance_response != []:
                relevance_response_ids = [
                    x["experiment_run_id"] for x in relevance_response
                ]
                relevance_scores = [
                    x["value"]["numericValue"] if "numericValue" in x["value"] else 0
                    for x in relevance_response
                ]
                relevance_df = pd.DataFrame(
                    {
                        "run_id": relevance_response_ids,
                        "relevance_score": relevance_scores,
                    }
                )
            else:
                relevance_df = pd.DataFrame(
                    {
                        "run_id": run_ids,
                        "relevance_score": pd.NA,
                    }
                )
            live_results_df = live_results_df.merge(
                relevance_df,
                on="run_id",
                how="left",
            )

            if context_relevancy_response != []:
                context_relevancy_response_ids = [
                    x["experiment_run_id"] for x in context_relevancy_response
                ]
                context_relevancy_scores = [
                    x["value"]["numericValue"] if "numericValue" in x["value"] else 0
                    for x in context_relevancy_response
                ]
                context_relevancy_df = pd.DataFrame(
                    {
                        "run_id": context_relevancy_response_ids,
                        "context_relevancy_score": context_relevancy_scores,
                    }
                )
            else:
                context_relevancy_df = pd.DataFrame(
                    {
                        "run_id": run_ids,
                        "context_relevancy_score": pd.NA,
                    }
                )
            live_results_df = live_results_df.merge(
                context_relevancy_df,
                on="run_id",
                how="left",
            )

            if maliciousness_response != []:
                maliciousness_response_ids = [
                    x["experiment_run_id"] for x in maliciousness_response
                ]
                maliciousness_scores = [
                    x["value"]["numericValue"] if "numericValue" in x["value"] else 0
                    for x in maliciousness_response
                ]
                maliciousness_df = pd.DataFrame(
                    {
                        "run_id": maliciousness_response_ids,
                        "maliciousness_score": maliciousness_scores,
                    }
                )
                maliciousness_df = maliciousness_df[
                    maliciousness_df["maliciousness_score"] != -1
                ]
            else:
                maliciousness_df = pd.DataFrame(
                    {
                        "run_id": run_ids,
                        "maliciousness_score": pd.NA,
                    }
                )
            live_results_df = live_results_df.merge(
                maliciousness_df,
                on="run_id",
                how="left",
            )
            if live_results_df["maliciousness_score"].isnull().all():
                live_results_df["maliciousness_score"] = 0.0

            if toxicity_response != []:
                toxicity_response_ids = [
                    x["experiment_run_id"] for x in toxicity_response
                ]
                toxicity_scores = [
                    x["value"]["numericValue"] if "numericValue" in x["value"] else 0
                    for x in toxicity_response
                ]
                toxicity_df = pd.DataFrame(
                    {
                        "run_id": toxicity_response_ids,
                        "toxicity_score": toxicity_scores,
                    }
                )
                toxicity_df = toxicity_df[toxicity_df["toxicity_score"] != -1]
            else:
                toxicity_df = pd.DataFrame(
                    {
                        "run_id": run_ids,
                        "toxicity_score": pd.NA,
                    }
                )
            live_results_df = live_results_df.merge(
                toxicity_df,
                on="run_id",
                how="left",
            )
            if live_results_df["toxicity_score"].isnull().all():
                live_results_df["toxicity_score"] = 0.0

            if comprehensiveness_response != []:
                comprehensiveness_response_ids = [
                    x["experiment_run_id"] for x in comprehensiveness_response
                ]
                comprehensiveness_scores = [
                    x["value"]["numericValue"] if "numericValue" in x["value"] else 0
                    for x in comprehensiveness_response
                ]
                comprehensiveness_df = pd.DataFrame(
                    {
                        "run_id": comprehensiveness_response_ids,
                        "comprehensiveness_score": comprehensiveness_scores,
                    }
                )
                comprehensiveness_df = comprehensiveness_df[
                    comprehensiveness_df["comprehensiveness_score"] != -1
                ]
            else:
                comprehensiveness_df = pd.DataFrame(
                    {
                        "run_id": run_ids,
                        "comprehensiveness_score": pd.NA,
                    }
                )
            live_results_df = live_results_df.merge(
                comprehensiveness_df,
                on="run_id",
                how="left",
            )
            if live_results_df["comprehensiveness_score"].isnull().all():
                live_results_df["comprehensiveness_score"] = 0.5

            with placeholder.container():

                kpi9, kpi10, kpi11, kpi12, kpi13 = st.columns([3, 3, 1, 1, 1])

                io_col, feedback_col = st.columns([3, 2])

                with io_col:
                    if "input_length" in live_results_df.columns:
                        avg_input_length = np.mean(live_results_df["input_length"])
                        input_lengths = live_results_df["input_length"].to_list()
                        kpi9.metric(
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

                    if "output_length" in live_results_df.columns:
                        avg_output_length = np.mean(live_results_df["output_length"])
                        output_lengths = live_results_df["output_length"].to_list()
                        kpi10.metric(
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

                    # Graphs
                    with st.expander(
                        ":material/input:/:material/output: **I/O Overview**",
                        expanded=True,
                    ):
                        fig_col4, fig_col5 = st.columns(2)

                        input_lengths_df = live_results_df[
                            ["input_length", "timestamp"]
                        ]
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
                            st.plotly_chart(
                                fig, key=f"input_length_fig_{update_timestamp}"
                            )

                        output_lengths_df = live_results_df[
                            ["output_length", "timestamp"]
                        ]
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
                kpi1, kpi2, kpi3, kpi4, kpi5, kpi6, kpi7 = st.columns(
                    [1, 1, 1, 1, 1, 1, 2]
                )

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
                                    live_results_df["relevance_score"]
                                    .to_list()
                                    .count(1),
                                    live_results_df["relevance_score"]
                                    .to_list()
                                    .count(0),
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
                        st.plotly_chart(
                            fig, key=f"maliciousness_fig_{update_timestamp}"
                        )

                    with fig_col5:
                        toxicity_df = live_results_df[
                            live_results_df["toxicity_score"].notnull()
                        ][["toxicity_score", "timestamp"]]
                        agg_toxicity_df = toxicity_df.groupby(
                            pd.Grouper(key="timestamp", freq="h")  # group by hour
                        )["toxicity_score"].agg(["mean", "max", "min"])
                        st.markdown(
                            "### Toxicity", help="Toxicity of the answer received."
                        )
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

                st.write("### Detailed Logs")
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
                live_results_df = live_results_df.drop(
                    columns=["response_id", "run_id"]
                )
                st.dataframe(
                    live_results_df.sort_values(by="timestamp", ascending=False)
                )
