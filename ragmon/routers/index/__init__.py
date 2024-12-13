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

import http
import json
import logging
import os
from pathlib import Path
import uuid
import sys
from typing import Dict, List, Optional, Union
import mlflow
import requests

import opentelemetry.trace
from fastapi import APIRouter
from llama_index.core.base.llms.types import MessageRole

from pydantic import BaseModel
from uvicorn.logging import DefaultFormatter

from ... import exceptions
from . import qdrant
from .qdrant import RagMessage
from ...config import settings
from ...data_types import CreateCustomEvaluatorRequest

logger = logging.getLogger(__name__)
formatter = DefaultFormatter("%(levelprefix)s %(message)s")

handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(formatter)

logger.addHandler(handler)
logger.setLevel(settings.rag_log_level)

tracer = opentelemetry.trace.get_tracer(__name__)

mlflow.set_tracking_uri(settings.mlflow.tracking_uri)
mlflow.llama_index.autolog()

router = APIRouter(
    prefix="/index",
    tags=["index"],
)


class RagPredictRequest(BaseModel):
    data_source_id: int
    chat_history: list[RagMessage]
    query: str
    configuration: qdrant.RagPredictConfiguration = qdrant.RagPredictConfiguration()
    do_evaluate: bool = True


class RagPredictSourceNode(BaseModel):
    node_id: str
    doc_id: str
    source_file_name: str
    score: float
    content: str


class RagPredictResponse(BaseModel):
    id: str
    input: str
    output: str
    source_nodes: List[RagPredictSourceNode] = []
    chat_history: list[RagMessage]
    mlflow_experiment_id: str
    mlflow_run_id: str


class MLflowStoreIdentifier(BaseModel):
    experiment_id: str
    experiment_run_id: str


class RagFeedbackRequest(BaseModel):
    experiment_id: str
    experiment_run_id: str
    feedback: float
    feedback_str: Optional[str] = None


@router.post("/add_custom_evaluator", summary="Add a custom evaluator")
@exceptions.propagates
@tracer.start_as_current_span("add_custom_evaluator")
def add_custom_evaluator(
    request: CreateCustomEvaluatorRequest,
) -> Dict[str, str]:
    """Add a custom evaluator"""
    try:
        path = Path(
            os.path.join(os.getcwd(), "evaluators", f"{request.data_source_id}")
        )
        path.mkdir(parents=True, exist_ok=True)
        save_to_disk(
            request.evaluator_definition,
            path,
            f"{request.evaluator_name.replace(" ", "_")}.json",
        )
        return {"status": "success"}
    except Exception as e:
        logger.error("Failed to add custom evaluator: %s", e)
        return {"status": "failed"}


@router.post("/feedback", summary="Log feedback for a response")
@exceptions.propagates
@tracer.start_as_current_span("feedback")
def feedback(
    request: RagFeedbackRequest,
) -> Dict[str, str]:
    """Log feedback for a response"""
    curr_exp = mlflow.set_experiment(experiment_id=request.experiment_id)
    try:
        with mlflow.start_run(
            experiment_id=curr_exp.experiment_id,
            run_id=request.experiment_run_id,
        ):
            mlflow.log_metrics(
                {
                    "feedback": request.feedback,
                },
                synchronous=False,
            )
            mlflow.log_table(
                {
                    "run_id": request.experiment_run_id,
                    "feedback_str": request.feedback_str,
                },
                artifact_file="user_feedback.json",
            )
            logger.info(
                "Logged feedback for exp id %s and run id %s",
                request.experiment_id,
                request.experiment_run_id,
            )
    except Exception as e:
        logger.error("Failed to log feedback: %s", e)
        return {"status": "failed"}
    return {"status": "success"}


def register_experiment_and_run(
    experiment_id: str,
    experiment_run_id: str,
) -> bool:
    try:
        mlflowstore = MLflowStoreIdentifier(
            experiment_id=experiment_id,
            experiment_run_id=experiment_run_id,
        )
        response = requests.post(
            url=f"{settings.mlflow_store.uri}/runs",
            data=mlflowstore.json(),
            headers={"Content-Type": "application/json"},
            timeout=10,
        )
        if response.status_code != http.HTTPStatus.OK:
            logger.error(
                "Failed to register experiment and run with MLflow store: %s",
                response.text,
            )
            return False
        logger.info(
            "Registered experiment id %s and run id %s with MLflow store",
            experiment_id,
            experiment_run_id,
        )
        return True
    except Exception as e:
        logger.error("Failed to register experiment and run with MLflow store: %s", e)
        return False


def save_to_disk(
    data,
    directory: Union[str, Path, os.PathLike],
    filename: str,
):
    """Helper function to save JSON data to disk."""
    with open(os.path.join(directory, filename), "w", encoding="utf-8") as f:
        json.dump(data, f)


@router.post("/predict", summary="Predict using indexed documents")
@exceptions.propagates
@tracer.start_as_current_span("predict")
async def predict(
    request: RagPredictRequest,
) -> RagPredictResponse:
    """Predict using indexed documents"""
    curr_exp = mlflow.set_experiment(experiment_name=f"{request.data_source_id}_live")
    with mlflow.start_run() as run:
        # deprecated: register experiment and run with MLflow store
        # register_experiment_and_run(
        #     experiment_id=curr_exp.experiment_id,
        #     experiment_run_id=run.info.run_id,
        # )

        # log request params
        mlflow.log_params(
            {
                "data_source_id": request.data_source_id,
                "top_k": request.configuration.top_k,
                "chunk_size": request.configuration.chunk_size,
                "model_name": request.configuration.model_name,
            }
        )
        response = qdrant.query(
            request.data_source_id,
            request.query,
            request.configuration,
            request.chat_history,
        )
        response_source_nodes = []
        for source_node in response.source_nodes:
            doc_id = os.path.basename(source_node.node.metadata["file_path"])
            response_source_nodes.append(
                RagPredictSourceNode(
                    node_id=source_node.node.node_id,
                    doc_id=doc_id,
                    source_file_name=source_node.node.metadata["file_name"],
                    score=source_node.score,
                    content=source_node.node.get_content(),
                )
            )
        response_source_nodes = sorted(
            response_source_nodes, key=lambda x: x.score, reverse=True
        )

        # add new messages and truncate to last 10
        new_history = request.chat_history + [
            RagMessage(role=MessageRole.USER, content=request.query),
            RagMessage(role=MessageRole.ASSISTANT, content=response.response),
        ]
        new_history = new_history[-10:]

        # TODO: maybe limit the number of source nodes returned
        rag_response = RagPredictResponse(
            id=str(uuid.uuid4()),
            input=request.query,
            output=response.response,
            source_nodes=response_source_nodes,
            chat_history=new_history,
            mlflow_experiment_id=curr_exp.experiment_id,
            mlflow_run_id=run.info.run_id,
        )

        if request.do_evaluate:
            # save response to disk for evaluation reconciler to pick up
            save_dir = Path(os.path.join(os.getcwd(), "data"))
            save_to_disk(
                rag_response.dict(),
                save_dir,
                f"{rag_response.id}.json",
            )

    return rag_response
