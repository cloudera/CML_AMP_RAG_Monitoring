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

import logging
from typing import Tuple

import opentelemetry.trace
import qdrant_client
from fastapi import HTTPException
from llama_index.core.base.llms.types import ChatMessage, MessageRole
from llama_index.core.chat_engine import CondenseQuestionChatEngine
from llama_index.core.evaluation import (
    FaithfulnessEvaluator,
    RelevancyEvaluator,
    ContextRelevancyEvaluator,
    EvaluationResult,
)

from llama_index.core.chat_engine.types import AgentChatResponse
from llama_index.core.indices import VectorStoreIndex
from llama_index.core.indices.vector_store import VectorIndexRetriever
from llama_index.core.node_parser import SentenceSplitter
from llama_index.core.query_engine import RetrieverQueryEngine
from llama_index.core.readers import SimpleDirectoryReader
from llama_index.core.response_synthesizers import get_response_synthesizer
from llama_index.core.storage import StorageContext
from llama_index.vector_stores.qdrant import QdrantVectorStore
from pydantic import BaseModel

from ...services.ragllm import get_embedding_model_and_dims, get_inference_model
from .judge import MaliciousnessEvaluator, ToxicityEvaluator, ComprehensivenessEvaluator
from pprint import pprint
import asyncio

logger = logging.getLogger(__name__)
tracer = opentelemetry.trace.get_tracer(__name__)


class RagMessage(BaseModel):
    role: MessageRole
    content: str


def table_name_from(data_source_id: int):
    return f"index_{data_source_id}"


class RagIndexDocumentConfiguration(BaseModel):
    # TODO: Add more params
    chunk_size: int = 512  # this is llama-index's default
    chunk_overlap: int = 10  # percentage of tokens in a chunk (chunk_size)


@tracer.start_as_current_span("qdrant upload")
def upload(
    tmpdirname: str,
    data_source_id: int,
    configuration: RagIndexDocumentConfiguration,
):
    try:
        documents = SimpleDirectoryReader(tmpdirname).load_data()
    except Exception as e:
        logger.error(
            "error loading document from temporary directory %s",
            tmpdirname,
        )
        raise HTTPException(
            status_code=422,
            detail=f"error loading document from temporary directory {tmpdirname}",
        ) from e

    logger.info("instantiating vector store")
    vector_store = create_qdrant_vector_store(data_source_id)
    logger.info("instantiated vector store")

    storage_context = StorageContext.from_defaults(vector_store=vector_store)

    chunk_overlap_tokens = int(
        configuration.chunk_overlap * 0.01 * configuration.chunk_size
    )

    embed_model, _ = get_embedding_model_and_dims()
    logger.info("indexing document")
    VectorStoreIndex.from_documents(
        documents,
        storage_context=storage_context,
        embed_model=embed_model,
        show_progress=False,
        transformations=[
            SentenceSplitter(
                chunk_size=configuration.chunk_size,
                chunk_overlap=chunk_overlap_tokens,
            ),
        ],
    )
    logger.info("indexed document")


class RagPredictConfiguration(BaseModel):
    top_k: int = 5
    chunk_size: int = 512
    model_name: str = "meta.llama3-70b-instruct-v1:0"


@tracer.start_as_current_span("Qdrant evaluate response")
async def evaluate_response(
    query: str,
    chat_response: AgentChatResponse,
) -> Tuple[
    EvaluationResult,
    EvaluationResult,
    EvaluationResult,
    EvaluationResult,
    EvaluationResult,
    EvaluationResult,
]:
    evaluator_llm = get_inference_model()

    relevancy_evaluator = RelevancyEvaluator(llm=evaluator_llm)
    faithfulness_evaluator = FaithfulnessEvaluator(llm=evaluator_llm)
    context_relevancy_evaluator = ContextRelevancyEvaluator(llm=evaluator_llm)
    maliciousness_evaluator = MaliciousnessEvaluator(llm=evaluator_llm)
    toxicity_evaluator = ToxicityEvaluator(llm=evaluator_llm)
    comprehensiveness_evaluator = ComprehensivenessEvaluator(llm=evaluator_llm)

    results = await asyncio.gather(
        relevancy_evaluator.aevaluate_response(query=query, response=chat_response),
        faithfulness_evaluator.aevaluate_response(query=query, response=chat_response),
        context_relevancy_evaluator.aevaluate_response(
            query=query, response=chat_response
        ),
        maliciousness_evaluator.aevaluate_response(query=query, response=chat_response),
        toxicity_evaluator.aevaluate_response(query=query, response=chat_response),
        comprehensiveness_evaluator.aevaluate_response(
            query=query, response=chat_response
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


@tracer.start_as_current_span("Qdrant query")
def query(
    data_source_id: int,
    query_str: str,
    configuration: RagPredictConfiguration,
    chat_history: list[RagMessage],
) -> AgentChatResponse:
    logger.info("fetching Qdrant index")
    try:
        vector_store = create_qdrant_vector_store(data_source_id)
    except Exception:
        # TODO: catch a more specific exception for index/namespace not found
        logger.error("Qdrant index or namespace not found")
        raise
    embed_model, _ = get_embedding_model_and_dims()
    index = VectorStoreIndex.from_vector_store(
        vector_store=vector_store,
        embed_model=embed_model,
    )
    logger.info("fetched Qdrant index")

    retriever = VectorIndexRetriever(
        index=index,
        similarity_top_k=configuration.top_k,
        embed_model=embed_model,
    )

    llm = get_inference_model()

    response_synthesizer = get_response_synthesizer(llm=llm)
    query_engine = RetrieverQueryEngine(
        retriever=retriever, response_synthesizer=response_synthesizer
    )
    chat_engine = CondenseQuestionChatEngine.from_defaults(
        query_engine=query_engine,
        llm=llm,
    )

    logger.info("querying chat engine")
    chat_history = list(
        map(
            lambda message: ChatMessage(role=message.role, content=message.content),
            chat_history,
        )
    )
    chat_response = chat_engine.chat(query_str, chat_history)
    logger.info("query response received from chat engine")
    pprint(chat_response)
    return chat_response


def create_qdrant_vector_store(data_source_id):
    client = qdrant_client.QdrantClient(host="localhost", port=6333)
    aclient = qdrant_client.AsyncQdrantClient(host="localhost", port=6334)
    vector_store = QdrantVectorStore(table_name_from(data_source_id), client, aclient)
    return vector_store
