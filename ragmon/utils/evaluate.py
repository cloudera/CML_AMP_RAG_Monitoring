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
This module contains functions to evaluate the response of a chat engine against a query and contexts.
"""

import asyncio
import json
import logging
import os
from pathlib import Path
import sys
from typing import Any, Union, Tuple, Sequence, Dict

from uvicorn.logging import DefaultFormatter

from llama_index.core.evaluation import (
    FaithfulnessEvaluator,
    RelevancyEvaluator,
    ContextRelevancyEvaluator,
    EvaluationResult,
)
from llama_index.llms.bedrock_converse import BedrockConverse

from .keyword_detection import extract_keywords
from .mlflowstore import register_experiment_and_run
from ..services.ragllm import get_inference_model
from ..data_types import RagPredictResponse, Metric
from llama_index.core.chat_engine.types import AgentChatResponse

import mlflow

from .judge import (
    MaliciousnessEvaluator,
    ToxicityEvaluator,
    ComprehensivenessEvaluator,
    load_custom_evaluator,
)
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


def get_custom_evaluators():
    # check for json files in custom evaluators directory
    if not CUSTOM_EVALUATORS_DIR.exists():
        return {}
    custom_evaluators = {}
    for file in CUSTOM_EVALUATORS_DIR.iterdir():
        if file.suffix == ".json":
            # read the json file
            eval_json = json.load(file.open())
            evaluator_name = eval_json.pop("name")
            custom_evaluators[evaluator_name] = eval_json
    return custom_evaluators


def convert_list_of_dicts_to_dict_of_lists(list_of_dicts: Sequence[Dict[str, Any]]):
    if not list_of_dicts:
        return {}

    keys = list_of_dicts[0].keys()
    dict_of_lists = {key: [] for key in keys}

    for item in list_of_dicts:
        for key, value in item.items():
            dict_of_lists[key].append(value)

    return dict_of_lists


async def evaluate_response(
    query: str,
    chat_response: Union[str, AgentChatResponse],
    contexts: Sequence[str] = None,
) -> Tuple[
    EvaluationResult,
    EvaluationResult,
    EvaluationResult,
    EvaluationResult,
    EvaluationResult,
    EvaluationResult,
    Dict[str, EvaluationResult],
]:
    """
    Evaluate a response against a query and contexts.

    Args:
        query: The query string.
        chat_response: The response to evaluate.
        contexts: The contexts to evaluate the response against.

    Returns:
        A tuple of evaluation results for the response.
    """
    evaluator_llm = get_inference_model()

    relevancy_evaluator = RelevancyEvaluator(llm=evaluator_llm)
    faithfulness_evaluator = FaithfulnessEvaluator(llm=evaluator_llm)
    context_relevancy_evaluator = ContextRelevancyEvaluator(llm=evaluator_llm)
    maliciousness_evaluator = MaliciousnessEvaluator(llm=evaluator_llm)
    toxicity_evaluator = ToxicityEvaluator(llm=evaluator_llm)
    comprehensiveness_evaluator = ComprehensivenessEvaluator(llm=evaluator_llm)

    if isinstance(chat_response, AgentChatResponse):
        results = await asyncio.gather(
            relevancy_evaluator.aevaluate_response(query=query, response=chat_response),
            faithfulness_evaluator.aevaluate_response(
                query=query, response=chat_response
            ),
            context_relevancy_evaluator.aevaluate_response(
                query=query, response=chat_response
            ),
            maliciousness_evaluator.aevaluate_response(
                query=query, response=chat_response
            ),
            toxicity_evaluator.aevaluate_response(query=query, response=chat_response),
            comprehensiveness_evaluator.aevaluate_response(
                query=query, response=chat_response
            ),
        )

    else:
        results = await asyncio.gather(
            relevancy_evaluator.aevaluate(
                query=query, response=chat_response, contexts=contexts
            ),
            faithfulness_evaluator.aevaluate(
                query=query, response=chat_response, contexts=contexts
            ),
            context_relevancy_evaluator.aevaluate(
                query=query, response=chat_response, contexts=contexts
            ),
            maliciousness_evaluator.aevaluate(
                query=query, response=chat_response, contexts=contexts
            ),
            toxicity_evaluator.aevaluate(
                query=query, response=chat_response, contexts=contexts
            ),
            comprehensiveness_evaluator.aevaluate(
                query=query, response=chat_response, contexts=contexts
            ),
        )

    (
        relevance,
        faithfulness,
        context_relevancy,
        maliciousness,
        toxicity,
        comprehensiveness,
    ) = results

    # check custom evaluators directory for custom evaluators
    custom_evaluators = get_custom_evaluators()
    loaded_custom_evals = []

    for _, evaluator_params in custom_evaluators.items():
        evaluator = load_custom_evaluator(
            eval_definition=evaluator_params["eval_definition"],
            questions=evaluator_params["questions"],
            llm=evaluator_llm,
        )
        loaded_custom_evals.append(evaluator)

    custom_eval_results = {}
    if loaded_custom_evals:
        custom_eval_results = await asyncio.gather(
            *[
                evaluator.aevaluate(
                    query=query, response=chat_response, contexts=contexts
                )
                for evaluator in loaded_custom_evals
            ]
        )

        custom_eval_results = {
            k: v for k, v in zip(custom_evaluators.keys(), custom_eval_results)
        }

    return (
        relevance,
        faithfulness,
        context_relevancy,
        maliciousness,
        toxicity,
        comprehensiveness,
        custom_eval_results,
    )


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

                # Evaluate the response
                (
                    relevance,
                    faithfulness,
                    context_relevancy,
                    maliciousness,
                    toxicity,
                    comprehensiveness,
                    custom_eval_results,
                ) = await evaluate_response(query, response, contexts)

                # show the evaluation results logger
                logger.info(
                    "Relevance: %s, Faithfulness: %s, "
                    "Context Relevancy: %s, Maliciousness: %s, "
                    "Toxicity: %s, Comprehensiveness: %s",
                    relevance.score,
                    faithfulness.score,
                    context_relevancy.score,
                    maliciousness.score,
                    toxicity.score,
                    comprehensiveness.score,
                )

                # show the custom evaluation metrics
                if custom_eval_results:
                    logger.info("Logging custom evaluation metrics")
                    for name, result in custom_eval_results.items():
                        logger.info("%s: %s", name, result.score)
                        mlflow.log_metric(
                            key=f"{name.lower().replace(' ', '_')}_score",
                            value=result.score,
                            synchronous=True,
                        )
                else:
                    logger.info("No custom evaluators or metrics to log")

                # create metric dictionary and do not add metrics which are none or empty

                metrics = [
                    Metric(name="relevance_score", value=relevance.score),
                    Metric(name="faithfulness_score", value=faithfulness.score),
                    Metric(
                        name="context_relevancy_score", value=context_relevancy.score
                    ),
                    Metric(name="maliciousness_score", value=maliciousness.score),
                    Metric(name="toxicity_score", value=toxicity.score),
                    Metric(
                        name="comprehensiveness_score", value=comprehensiveness.score
                    ),
                    Metric(name="input_length", value=len(query.split())),
                    Metric(name="output_length", value=len(response.split())),
                ]

                # add custom metrics to metrics list
                for name, result in custom_eval_results.items():
                    metrics.append(
                        Metric(
                            name=f"{name.lower().replace(' ', '_')}_score",
                            value=result.score,
                        )
                    )

                data.metrics = metrics

                # Log the metrics
                for metric in metrics:
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
                    "query_keywords": ", ".join(query_keywords or []),
                    "response_keywords": ", ".join(response_keywords or []),
                }

                mlflow.log_table(
                    response_table,
                    artifact_file="live_results.json",
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
