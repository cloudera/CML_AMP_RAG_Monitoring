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

import json
import logging
import os
from pathlib import Path
import time
import uuid
import sys
from typing import Dict
import opentelemetry.trace
from fastapi import APIRouter
from llama_index.core.base.llms.types import MessageRole
from llama_index.core.chat_engine.types import AgentChatResponse

from uvicorn.logging import DefaultFormatter

from ... import exceptions
from . import qdrant
from ...utils import save_to_disk
from ...data_types import (
    RagPredictRequest,
    RagPredictSourceNode,
    RagPredictResponse,
    RagFeedbackRequest,
    RagFeedback,
    RagMessage,
)
from ...config import settings
from ...data_types import CreateCustomEvaluatorRequest

from pprint import pprint

logger = logging.getLogger(__name__)
formatter = DefaultFormatter("%(levelprefix)s %(message)s")

handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(formatter)

logger.addHandler(handler)
logger.setLevel(settings.rag_log_level)

tracer = opentelemetry.trace.get_tracer(__name__)

router = APIRouter(
    prefix="/index",
    tags=["index"],
)


@router.post("/add_custom_evaluator", summary="Add a custom evaluator")
@exceptions.propagates
@tracer.start_as_current_span("add_custom_evaluator")
def add_custom_evaluator(
    request: CreateCustomEvaluatorRequest,
) -> Dict[str, str]:
    """Add a custom evaluator"""
    try:
        data_directory = os.path.join(os.getcwd(), "data")
        custom_evals_dir = Path(os.path.join(data_directory, "custom_evaluators"))
        custom_evals_dir.mkdir(parents=True, exist_ok=True)
        save_to_disk(
            request.dict(),
            custom_evals_dir,
            f"{request.name.lower().replace(' ', '_')}.json",
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
    # Open response file
    if request.response_id:
        data_directory = os.path.join(os.getcwd(), "data")
        response_directory = os.path.join(data_directory, "responses")
        response_file = os.path.join(response_directory, f"{request.response_id}.json")
        # check if file exists
        if not os.path.exists(response_file):
            raise FileNotFoundError(f"Response file {response_file} not found")
        with open(response_file, "r") as f:
            response_data = json.load(f)
        response = RagPredictResponse(**response_data)

        # Log feedback
        response.feedback = RagFeedback(
            feedback=request.feedback,
            feedback_str=request.feedback_str,
        )

        response.feedback_logged_status = "pending"

        # save response to disk for evaluation reconciler to pick up
        save_to_disk(
            data=response.dict(),
            directory=response_directory,
            filename=f"{response.id}.json",
        )

        logger.info("Feedback queued for response %s", request.response_id)

        return {"status": "success"}


@router.post("/predict", summary="Predict using indexed documents")
@exceptions.propagates
@tracer.start_as_current_span("predict")
async def predict(
    request: RagPredictRequest,
) -> RagPredictResponse:
    """Predict using indexed documents"""
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
    pprint(new_history)

    # TODO: maybe limit the number of source nodes returned
    rag_response = RagPredictResponse(
        id=str(uuid.uuid4()),
        input=request.query,
        output=response.response,
        data_source_id=request.data_source_id,
        top_k=request.configuration.top_k,
        chunk_size=request.configuration.chunk_size,
        model_name=request.configuration.model_name,
        source_nodes=response_source_nodes,
        chat_history=new_history,
        metrics_logged_status="pending",
        timestamp=time.strftime("%Y-%m-%d %H:%M:%S"),
    )

    if request.do_evaluate:
        # save response to disk for evaluation reconciler to pick up
        data_directory = os.path.join(os.getcwd(), "data")
        response_directory = os.path.join(data_directory, "responses")
        save_to_disk(
            data=rag_response.dict(),
            directory=response_directory,
            filename=f"{rag_response.id}.json",
        )

    return rag_response
