"""
This module contains functions to evaluate the response of a chat engine against a query and contexts.
"""

import asyncio
import logging
import sys
from typing import Union, Tuple, Sequence

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

from .judge import MaliciousnessEvaluator, ToxicityEvaluator, ComprehensivenessEvaluator
from ..config import settings

logger = logging.getLogger(__name__)
formatter = DefaultFormatter("%(levelprefix)s %(message)s")

handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(formatter)

logger.addHandler(handler)
logger.setLevel(settings.rag_log_level)


def table_name_from(data_source_id: int):
    return f"index_{data_source_id}"


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

    return (
        relevance,
        faithfulness,
        context_relevancy,
        maliciousness,
        toxicity,
        comprehensiveness,
    )


async def evaluate_json_data(data):
    """
    Evaluate the JSON data and log the evaluation metrics to MLflow

    Args:
        data: The JSON data to evaluate.

    Returns:
        data: The evaluated JSON data.
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
            # set the experiment ID and run ID if not present
            mlflow_experiment = mlflow.set_experiment(
                experiment_name=f"{data.data_source_id}_live"
            )
            data.mlflow_experiment_id = mlflow_experiment.experiment_id
            if data.mlflow_run_id:
                run = mlflow.start_run(
                    run_id=data.mlflow_run_id,
                )
            else:
                run = mlflow.start_run()
                data.mlflow_run_id = run.info.run_id

            register_experiment_and_run(
                experiment_id=data.mlflow_experiment_id,
                experiment_run_id=data.mlflow_run_id,
            )

            # log request params
            mlflow.log_params(
                {
                    "data_source_id": data_source_id,
                    "top_k": top_k,
                    "chunk_size": chunk_size,
                    "model_name": model_name,
                }
            )

            # log contexts
            if contexts:
                for i, context in enumerate(contexts):
                    mlflow.log_param(f"context_{i}", context)

            # Evaluate the response
            (
                relevance,
                faithfulness,
                context_relevancy,
                maliciousness,
                toxicity,
                comprehensiveness,
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

            # create metric dictionary and do not add metrics which are none or empty

            metrics = [
                Metric(name="relevance_score", value=relevance.score),
                Metric(name="faithfulness_score", value=faithfulness.score),
                Metric(name="context_relevancy_score", value=context_relevancy.score),
                Metric(name="maliciousness_score", value=maliciousness.score),
                Metric(name="toxicity_score", value=toxicity.score),
                Metric(name="comprehensiveness_score", value=comprehensiveness.score),
                Metric(name="input_length", value=len(query.split())),
                Metric(name="output_length", value=len(response.split())),
            ]

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

            # log response
            mlflow.log_table(
                {
                    "response_id": data.id,
                    "input": query,
                    "input_length": len(query.split()),
                    "output": response,
                    "output_length": len(response.split()),
                    "source_nodes": data.source_nodes,
                    "query_keywords": ", ".join(query_keywords or []),
                    "response_keywords": ", ".join(response_keywords or []),
                },
                artifact_file="live_results.json",
            )

            logger.info(
                "Logged keywords for exp id %s and run id %s",
                data.mlflow_experiment_id,
                data.mlflow_run_id,
            )

            mlflow.end_run()

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
                run = mlflow.start_run(
                    experiment_id=data.mlflow_experiment_id,
                    run_id=data.mlflow_run_id,
                )
                mlflow.log_metrics(
                    {
                        "feedback": data.feedback.feedback,
                    },
                    synchronous=False,
                )
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
                mlflow.end_run()
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
