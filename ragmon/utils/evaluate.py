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

"""
Module for functions to evaluate the response against a query and contexts.
"""

import sys
import asyncio
import json
import logging
import os
from pathlib import Path
from typing import Any, Optional, Union, Tuple, Sequence, Dict

import mlflow
from uvicorn.logging import DefaultFormatter
from llama_index.core.chat_engine.types import AgentChatResponse
from llama_index.core.evaluation import (
    FaithfulnessEvaluator,
    RelevancyEvaluator,
    ContextRelevancyEvaluator,
    EvaluationResult,
    BaseEvaluator,
)

from .keyword_detection import extract_keywords
from .mlflowstore import register_experiment_and_run
from ..services.ragllm import get_inference_model
from ..data_types import RagPredictResponse, Metric
from .judge import (
    MaliciousnessEvaluator,
    ToxicityEvaluator,
    ComprehensivenessEvaluator,
    load_custom_evaluator,
)
from .custom_evals import get_custom_evaluators
from ..config import settings

logger = logging.getLogger(__name__)
formatter = DefaultFormatter("%(levelprefix)s %(message)s")

handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(formatter)

logger.addHandler(handler)
logger.setLevel(settings.rag_log_level)

# custom evaluators directory
main_dir = Path(os.path.realpath(__file__)).parents[1]
data_dir = Path(os.path.join(main_dir, "data"))
CUSTOM_EVALUATORS_DIR = Path(os.path.join(data_dir, "custom_evaluators"))


def convert_list_of_dicts_to_dict_of_lists(list_of_dicts: Sequence[Dict[str, Any]]):
    """
    Convert a list of dictionaries to a dictionary of lists.

    Args:
        list_of_dicts (Sequence[Dict[str, Any]]): The list of dictionaries to convert.

    Returns:
        Dict[str, List[Any]]: The dictionary of lists.
    """
    if not list_of_dicts:
        return {}

    keys = list_of_dicts[0].keys()
    dict_of_lists = {key: [] for key in keys}

    for item in list_of_dicts:
        for key, value in item.items():
            dict_of_lists[key].append(value)

    return dict_of_lists


def load_all_evaluators(exp_id: Optional[str] = None) -> Dict[str, BaseEvaluator]:
    """
    Load all evaluators including custom evaluators.

    Args:
        exp_id (Optional[str], optional): The experiment ID. Defaults to None.

    Returns:
        Dict[str, Dict[str, Any]]: The loaded custom evaluators.
    """
    # load
    evaluator_llm = get_inference_model()
    evaluators = {}
    custom_evals = get_custom_evaluators(exp_id)

    evaluators["relevancy"] = RelevancyEvaluator(llm=evaluator_llm)
    evaluators["faithfulness"] = FaithfulnessEvaluator(llm=evaluator_llm)
    evaluators["context_relevancy"] = ContextRelevancyEvaluator(llm=evaluator_llm)
    evaluators["maliciousness"] = MaliciousnessEvaluator(llm=evaluator_llm)
    evaluators["toxicity"] = ToxicityEvaluator(llm=evaluator_llm)
    evaluators["comprehensiveness"] = ComprehensivenessEvaluator(llm=evaluator_llm)

    for _, evaluator_params in custom_evals.items():
        evaluator = load_custom_evaluator(
            eval_definition=evaluator_params["eval_definition"],
            questions=evaluator_params["questions"],
            llm=evaluator_llm,
        )
        evaluators[evaluator_params["name"].lower()] = evaluator

    return evaluators


def set_experiment_and_run(data):
    """
    Set the experiment ID and run ID for MLflow.

    Args:
        data: The JSON data containing the experiment ID and run ID.

    Returns:
        Tuple containing the experiment ID and run ID.
    """
    if data.mlflow_experiment_id is None or data.mlflow_run_id is None:
        # set the experiment ID and run ID if not present
        mlflow_experiment = mlflow.set_experiment(
            experiment_name=f"{data.data_source_id}_live"
        )
        mlflow_experiment_id = mlflow_experiment.experiment_id
        if data.mlflow_run_id:
            run = mlflow.start_run(
                run_id=data.mlflow_run_id,
            )
        else:
            run = mlflow.start_run()
        mlflow_run_id = run.info.run_id
        logger.info(
            "Set experiment ID %s and run ID %s first time for response with id %s",
            mlflow_experiment_id,
            mlflow_run_id,
            data.id,
        )
        if mlflow.active_run():
            mlflow.end_run()

        return mlflow_experiment_id, mlflow_run_id
    else:
        logger.info(
            "Experiment ID %s and run ID %s already set for response with id %s",
            data.mlflow_experiment_id,
            data.mlflow_run_id,
            data.id,
        )
        return data.mlflow_experiment_id, data.mlflow_run_id


async def evaluate_json_data(data):
    """
    Evaluate the JSON data and log the evaluation metrics to MLflow

    Args:
        data: The JSON data to evaluate.

    Returns:
        Dictionary containing the status and the evaluated JSON data.
            data: The evaluated JSON data.
            status: The status of the evaluation.
    """
    data = RagPredictResponse(**data)
    if (
        data.metrics_logged_status == "success"
        and data.feedback_logged_status == "success"
    ):
        return {
            "data": data.dict(),
            "status": "success",
        }

    if data.mlflow_experiment_id is None or data.mlflow_run_id is None:
        data.mlflow_experiment_id, data.mlflow_run_id = set_experiment_and_run(data)
        return {
            "data": data.dict(),
            "status": "pending",
        }

    if data.metrics_logged_status == "pending":
        logger.info("Evaluating response with id %s", data.id)
        response_id = data.id
        query = data.input
        response = data.output
        data_source_id = data.data_source_id
        top_k = data.top_k
        chunk_size = data.chunk_size
        model_name = data.model_name
        contexts = []
        for source_node in data.source_nodes:
            contexts.append(source_node.content)
        try:
            register_experiment_and_run(
                experiment_id=data.mlflow_experiment_id,
                experiment_run_id=data.mlflow_run_id,
            )
            with mlflow.start_run(
                experiment_id=data.mlflow_experiment_id,
                run_id=data.mlflow_run_id,
            ):
                # log request params
                mlflow.log_params(
                    {
                        "data_source_id": data_source_id,
                        "top_k": top_k,
                        "chunk_size": chunk_size,
                        "model_name": model_name,
                    }
                )

                # load all evaluators
                evaluators = load_all_evaluators(data.mlflow_experiment_id)

                # Evaluate the response
                eval_results = await asyncio.gather(
                    *[
                        evaluator.aevaluate(query, response, contexts)
                        for evaluator in evaluators.values()
                    ]
                )

                eval_results_dict = {
                    evaluator_name: eval_result
                    for evaluator_name, eval_result in zip(
                        evaluators.keys(), eval_results
                    )
                }

                # log the evaluation results
                data.metrics = []

                for evaluator_name, eval_result in eval_results_dict.items():
                    if isinstance(eval_result, EvaluationResult):
                        data.metrics.append(
                            Metric(
                                name=f"{evaluator_name}_score",
                                value=eval_result.score,
                            )
                        )

                # Log the metrics
                for metric in data.metrics:
                    if metric.value is not None:
                        mlflow.log_metric(
                            metric.name,
                            metric.value,
                            synchronous=False,
                        )

                logger.info(
                    "Logged evaluation metrics for exp id %s and run id %s",
                    data.mlflow_experiment_id,
                    data.mlflow_run_id,
                )

                # extract keywords from the query and response
                query_keywords = extract_keywords(query)
                response_keywords = extract_keywords(response)

                # store response in dict to log
                response_table = {
                    "timestamp": data.timestamp,
                    "response_id": data.id,
                    "input": query,
                    "output": response,
                }

                mlflow.log_table(
                    response_table,
                    artifact_file="live_results.json",
                )

                # log the keywords
                mlflow.log_table(
                    {
                        "query_keywords": ", ".join(query_keywords or []),
                        "response_keywords": ", ".join(response_keywords or []),
                    },
                    artifact_file="keywords.json",
                )

                # log the source nodes/contexts
                source_nodes = [
                    node.model_dump(mode="json") for node in data.source_nodes
                ]
                source_nodes_dict = convert_list_of_dicts_to_dict_of_lists(source_nodes)
                mlflow.log_table(
                    source_nodes_dict,
                    artifact_file="source_nodes.json",
                )

                logger.info(
                    "Logged response and keywords for exp id %s and run id %s",
                    data.mlflow_experiment_id,
                    data.mlflow_run_id,
                )

            data.metrics_logged_status = "success"

        except Exception as e:
            logger.error(
                "Failed to log evaluation metrics for response with id %s with error: %s",
                response_id,
                e,
            )
            if mlflow.active_run():
                mlflow.end_run()

    if data.feedback_logged_status == "pending":
        if data.mlflow_experiment_id and data.mlflow_run_id:
            try:
                logger.info("Logging feedback for response with id %s", data.id)
                with mlflow.start_run(
                    experiment_id=data.mlflow_experiment_id,
                    run_id=data.mlflow_run_id,
                ):
                    if data.feedback.feedback is not None:
                        mlflow.log_metrics(
                            {
                                "feedback": data.feedback.feedback,
                            },
                            synchronous=False,
                        )
                    if data.feedback.feedback_str:
                        mlflow.log_table(
                            {
                                "feedback_str": data.feedback.feedback_str,
                            },
                            artifact_file="user_feedback.json",
                        )
                    logger.info(
                        "Logged feedback for exp id %s and run id %s",
                        data.mlflow_experiment_id,
                        data.mlflow_run_id,
                    )
                data.feedback_logged_status = "success"
            except Exception as e:
                logger.error(
                    "Failed to log feedback for response with id %s with error: %s",
                    data.id,
                    e,
                )
                if mlflow.active_run():
                    mlflow.end_run()
        if (
            data.feedback_logged_status == "pending"
            and data.metrics_logged_status == "pending"
        ):
            logger.error(
                "Failed to log feedback and metrics for response with id %s",
                data.id,
            )
            return {
                "data": data.dict(),
                "status": "failed",
            }

        return {
            "data": data.dict(),
            "status": "success",
        }
