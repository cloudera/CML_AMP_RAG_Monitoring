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

from .keyword import extract_keywords
from ..services.ragllm import get_inference_model
from llama_index.core.chat_engine.types import AgentChatResponse

import mlflow
from mlflow.tracking import MlflowClient

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
        A dictionary containing the status, metrics, and error
    """
    mlflow_experiment_id = data["mlflow_experiment_id"]
    mlflow_run_id = data["mlflow_run_id"]
    response_id = data["id"]
    query = data["input"]
    response = data["output"]
    contexts = []
    if "source_nodes" in data:
        for source_node in data["source_nodes"]:
            contexts.append(source_node["content"])

    try:
        # set the experiment ID and run ID
        if mlflow_experiment_id:
            mlflow.set_experiment(experiment_id=mlflow_experiment_id)
        else:
            mlflow_experiment_id = mlflow.create_experiment(
                name=f"{data['data_source_id']}_live"
            )
            data["mlflow_experiment_id"] = mlflow_experiment_id
        if mlflow_run_id:
            run = mlflow.start_run(
                experiment_id=mlflow_experiment_id,
                run_id=mlflow_run_id,
            )
        else:
            run = mlflow.start_run(experiment_id=mlflow_experiment_id)
            data["mlflow_run_id"] = run.info.run_id
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

        # Log the evaluation results
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
        # fetch previous metrics

        # create metric dictionary and do not add metrics which are none or empty
        metrics = {
            "relevance_score": relevance.score,
            "faithfulness_score": faithfulness.score,
            "context_relevancy_score": context_relevancy.score,
            "maliciousness_score": maliciousness.score,
            "toxicity_score": toxicity.score,
            "comprehensiveness_score": comprehensiveness.score,
            "output_length": len(response.split()),
            "input_length": len(query.split()),
        }

        # Log the metrics
        for key, value in metrics.items():
            if value is not None:
                mlflow.log_metric(
                    key,
                    value,
                    synchronous=False,
                )

        logger.info(
            "Logged evaluation metrics for exp id %s and run id %s",
            mlflow_experiment_id,
            mlflow_run_id,
        )

        # extract keywords from the query and response
        query_keywords = extract_keywords(query)
        response_keywords = extract_keywords(response)

        # log response
        mlflow.log_table(
            {
                "response_id": data["id"],
                "input": query,
                "input_length": len(query.split()),
                "output": response,
                "output_length": len(response.split()),
                "source_nodes": data["source_nodes"],
                "query_keywords": ", ".join(query_keywords or []),
                "response_keywords": ", ".join(response_keywords or []),
            },
            artifact_file="live_results.json",
        )

        logger.info(
            "Logged keywords for exp id %s and run id %s",
            mlflow_experiment_id,
            mlflow_run_id,
        )

        mlflow.end_run()

        data["metrics_logged_status"] = "success"

        return {"status": "success", "metrics": metrics, "data": data, "error": None}
    except Exception as e:
        logger.error(
            "Failed to log evaluation metrics for response with id %s with error: %s",
            response_id,
            e,
        )
        data["metrics_logged_status"] = "failed"
        return {"status": "failed", "metrics": None, "data": data, "error": str(e)}
