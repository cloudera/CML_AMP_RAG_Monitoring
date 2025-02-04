import json
import os
import shutil
import sys
from pathlib import Path
from typing import List
from tqdm import tqdm

import pandas as pd
import requests
from llama_index.core import Settings
from llama_index.core.base.llms.types import MessageRole
from llama_index.core.indices import VectorStoreIndex
from llama_index.core.node_parser import SentenceSplitter
from llama_index.core.readers import SimpleDirectoryReader
from llama_index.core.storage import StorageContext
from llama_index.embeddings.bedrock import BedrockEmbedding
from llama_index.vector_stores.qdrant import QdrantVectorStore
from pydantic import BaseModel
from qdrant_client import QdrantClient
from qdrant_client.models import Distance, VectorParams

# Add the st_app directory to the sys.path
file_path = Path(os.path.realpath(__file__))
main_dir = file_path.parents[1]
st_app_dir = os.path.join(main_dir, "st_app")
sys.path.append(st_app_dir)

from data_types import (
    RagFeedbackRequest,
    RagIndexConfiguration,
    RagPredictRequest,
    RagPredictResponse,
)

COLLECTIONS_JSON = os.path.join(st_app_dir, "collections.json")
SOURCE_FILES_DIR = os.path.join(st_app_dir, "source_files")
SAMPLE_DATA_DIR = os.path.join(main_dir, "sample_data")
CUSTOM_EVAL_DIR = os.path.join(main_dir, "custom_evaluators")

# Create the source files directory
os.makedirs(SOURCE_FILES_DIR, exist_ok=True)

Settings.embed_model = BedrockEmbedding(model_name="cohere.embed-english-v3")
EMBED_DIMS = 1024


# Function to get the table name from a data source ID
def table_name_from(data_source_id: int):
    """Get the table name from a data source ID."""
    return f"index_{data_source_id}"


def get_response(request: RagPredictRequest) -> RagPredictResponse:
    """Get a response from the RAG model."""
    fastapi_port = os.environ["FASTAPI_PORT"]
    if fastapi_port is None:
        fastapi_port = 8000

    response = requests.post(
        url=f"http://localhost:{fastapi_port}/index/predict",
        data=request.json(),
        headers={
            "Content-Type": "application/json",
            "Accept": "application/json",
        },
        timeout=60,
    )

    if response.status_code != 200:
        raise ValueError(f"Failed to get response: {response.text}")

    return RagPredictResponse(**response.json())


# Function to log feedback
def log_feedback(request: RagFeedbackRequest):
    """Logs feedback for a given response."""
    fastapi_port = os.environ["FASTAPI_PORT"]
    if fastapi_port is None:
        fastapi_port = 8000
    response = requests.post(
        url=f"http://localhost:{fastapi_port}/index/feedback",
        data=request.json(),
        headers={
            "Content-Type": "application/json",
            "Accept": "application/json",
        },
        timeout=60,
    )

    if response.status_code != 200:
        raise ValueError(f"Failed to log feedback: {response.text}")

    return response.json()


def copy_files(src_dir, dest_dir):
    """Copies all files from src_dir to dest_dir."""

    for filename in os.listdir(src_dir):
        src_file = os.path.join(src_dir, filename)
        dest_file = os.path.join(dest_dir, filename)

        if os.path.isfile(src_file):
            shutil.copy2(src_file, dest_file)


def main():
    # Create a collections.json file and write an empty list to it
    if not os.path.exists(COLLECTIONS_JSON):
        with open(COLLECTIONS_JSON, "w+") as f:
            collections = []
            json.dump(collections, f)

    # Read the collections.json file
    with open(COLLECTIONS_JSON, "r+") as f:
        try:
            collections = json.load(f)
        except json.JSONDecodeError:
            collections = []

    print("Creating a Qdrant vector store...")

    # Function to get or create a Qdrant vector store
    client = QdrantClient(host="localhost", port=6333)
    collection_config = {
        "id": len(collections) + 1,
        "name": "CML Docs",
        "vector_size": EMBED_DIMS,
        "distance_metric": "Cosine",
        "chunk_size": 512,
        "chunk_overlap": 0,
    }
    index_config = RagIndexConfiguration(**collection_config)
    client = QdrantClient(url="http://localhost:6333")
    client.create_collection(
        collection_name=table_name_from(index_config.id),
        vectors_config=VectorParams(
            size=index_config.vector_size, distance=Distance.COSINE
        ),
    )
    print("Created a Qdrant vector store.")

    # Write the collection configuration to the collections.json file
    collections.append(collection_config)
    with open(COLLECTIONS_JSON, "w+") as f:
        json.dump(collections, f)
    print("Wrote the collection configuration to collections.json.")

    # Create a directory for the index
    index_dir = os.path.join(SOURCE_FILES_DIR, table_name_from(index_config.id))
    os.makedirs(
        index_dir,
        exist_ok=True,
    )

    # Copy the sample data to the index directory
    copy_files(SAMPLE_DATA_DIR, index_dir)

    # Populate the Qdrant vector store
    print("Populating the Qdrant vector store...")
    vector_store = QdrantVectorStore(
        table_name_from(index_config.id),
        client,
        dense_config=VectorParams(
            size=index_config.vector_size, distance=Distance.COSINE
        ),
    )
    documents = SimpleDirectoryReader(
        index_dir,
    ).load_data()
    storage_context = StorageContext.from_defaults(vector_store=vector_store)
    chunk_overlap_tokens = int(
        index_config.chunk_overlap * 0.01 * index_config.chunk_size
    )
    VectorStoreIndex.from_documents(
        documents,
        storage_context=storage_context,
        show_progress=False,
        transformations=[
            SentenceSplitter(
                chunk_size=index_config.chunk_size,
                chunk_overlap=chunk_overlap_tokens,
            )
        ],
    )
    print("Populated the Qdrant vector store.")

    print("Creating a custom evaluator...")

    # Create a custom evaluator
    friendliness_eval_def = {
        "name": "Friendliness",
        "eval_definition": "Friendliness assesses the warmth and approachability of the answer.",
        "questions": "Is the answer friendly?\nIs the answer compassionate?",
        "examples": [],
    }

    if not os.path.exists(CUSTOM_EVAL_DIR):
        os.makedirs(CUSTOM_EVAL_DIR)
    with open(os.path.join(CUSTOM_EVAL_DIR, "friendliness.json"), "w+") as f:
        json.dump(friendliness_eval_def, f, indent=2)

    # simulate question answering
    print("Simulating question answering...")
    questions = [
        "How does Cloudera Machine Learning work?",
        "What is the difference between Cloudera Machine Learning and Cloudera Data Science Workbench?",
        "How can I train models?",
        "Does CML support GPUs?",
        "does auto-scaling of endpoints work in CML?",
    ]
    for i, question in tqdm(enumerate(questions)):
        response = get_response(
            RagPredictRequest(
                data_source_id=index_config.id,
                query=question,
                chat_history=[],  # chat history is not needed for this simulation
            )
        )
        feedback_request = RagFeedbackRequest(
            experiment_id=response.mlflow_experiment_id,
            experiment_run_id=response.mlflow_run_id,
            feedback=i % 2,
        )
        log_feedback(feedback_request)
    print("Simulated question answering and evaluation of responses.")


if __name__ == "__main__":
    main()
