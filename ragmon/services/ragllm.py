#
#  CLOUDERA APPLIED MACHINE LEARNING PROTOTYPE (AMP)
#  (C) Cloudera, Inc. 2024
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
#  Absent a written agreement with Cloudera, Inc. ("Cloudera") to the
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
import os

from llama_index.core.base.embeddings.base import BaseEmbedding
from llama_index.core.llms import LLM
from llama_index.embeddings.bedrock import BedrockEmbedding
from llama_index.llms.bedrock_converse import BedrockConverse

from .llama_utils import messages_to_prompt, completion_to_prompt
from .CaiiEmbeddingModel import CaiiEmbeddingModel
from .caii import describe_endpoint, get_llm

caii_domain: str = os.environ.get("CAII_DOMAIN", "")


def is_caii_enabled() -> bool:
    return len(caii_domain) > 0


def get_inference_model_name() -> str:
    if is_caii_enabled():
        # CAII
        return os.getenv("CAII_INFERENCE_ENDPOINT_NAME", "")

    # Bedrock
    return "meta.llama3-70b-instruct-v1:0"


def get_embedding_model_name() -> str:
    if is_caii_enabled():
        return os.getenv("CAII_EMBEDDING_ENDPOINT_NAME", "")

    # Bedrock
    return "cohere.embed-english-v3"


def get_caii_embedding_model() -> BaseEmbedding:
    endpoint = describe_endpoint(
        domain=caii_domain, endpoint_name=get_embedding_model_name()
    )
    return CaiiEmbeddingModel(endpoint=endpoint)


def get_embedding_model_and_dims() -> (BaseEmbedding, int):
    if is_caii_enabled():
        return get_caii_embedding_model(), 4096
    else:
        return BedrockEmbedding(model_name=get_embedding_model_name()), 1024


def get_inference_model() -> LLM:
    if is_caii_enabled():
        return get_llm(
            domain=caii_domain,
            endpoint_name=get_inference_model_name(),
            messages_to_prompt=messages_to_prompt,
            completion_to_prompt=completion_to_prompt,
        )
    else:
        return BedrockConverse(model=get_inference_model_name())
