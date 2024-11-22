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

from pydantic import BaseModel
from llama_index.core.base.llms.types import MessageRole
from typing import List, Optional


class RagIndexConfiguration(BaseModel):
    id: int
    name: str
    vector_size: int = 1024
    distance_metric: str = "Cosine"
    chunk_size: int = 512
    chunk_overlap: int = 10


class RagPredictConfiguration(BaseModel):
    top_k: int = 5
    chunk_size: int = 512
    model_name: str = "meta.llama3-70b-instruct-v1:0"


class RagPredictSourceNode(BaseModel):
    node_id: str
    doc_id: str
    source_file_name: str
    score: float
    content: str


class RagMessage(BaseModel):
    role: MessageRole
    content: str


class RagIndexDocumentConfiguration(BaseModel):
    # TODO: Add more params
    chunk_size: int = 512  # this is llama-index's default
    chunk_overlap: int = 10  # percentage of tokens in a chunk (chunk_size)


class RagPredictRequest(BaseModel):
    # session_id: str
    data_source_id: int
    chat_history: List[RagMessage]
    query: str
    configuration: RagPredictConfiguration = RagPredictConfiguration()
    do_evaluate: bool = True


class RagPredictResponse(BaseModel):
    id: str
    input: str
    output: str
    source_nodes: List[RagPredictSourceNode] = []
    chat_history: List[RagMessage]
    mlflow_experiment_id: str
    mlflow_run_id: str


class RagFeedbackRequest(BaseModel):
    experiment_id: str
    experiment_run_id: str
    feedback: float  # 0.0 or 1.0\
    feedback_str: Optional[str] = None


class MLFlowStoreRequest(BaseModel):
    experiment_id: str
    run_ids: List[str] = []
    metric_names: List[str] = []
