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
import os
import time
from pathlib import Path
import pandas as pd
import streamlit as st
from qdrant_client import QdrantClient
from llama_index.core import Settings
from llama_index.core.indices import VectorStoreIndex
from llama_index.core.node_parser import SentenceSplitter
from llama_index.core.readers import SimpleDirectoryReader
from llama_index.core.storage import StorageContext
from llama_index.vector_stores.qdrant import QdrantVectorStore
from qdrant_client.models import Distance, VectorParams
from services.ragllm import get_embedding_model_and_dims
from data_types import RagIndexConfiguration
import mimetypes


Settings.embed_model, EMBED_DIMS = get_embedding_model_and_dims()

# get resources directory
file_path = Path(os.path.realpath(__file__))
st_app_dir = file_path.parents[1]
RESOURCES_DIR = os.path.join(st_app_dir, "resources")
COLLECTIONS_JSON = os.path.join(st_app_dir, "collections.json")
SOURCE_FILES_DIR = os.path.join(st_app_dir, "source_files")


# Function to get list of collections
def get_collections():
    """
    Retrieve a list of collections from the client.
    Returns:
        list: A list of collections retrieved from the client.
    """
    client = QdrantClient(url="http://localhost:6333")
    collections = client.get_collections().collections
    if len(collections) == 0:
        with open(COLLECTIONS_JSON, "w+") as f:
            collections = []
            json.dump(collections, f)
    else:
        with open(COLLECTIONS_JSON, "r+") as f:
            try:
                collections = json.load(f)
            except json.JSONDecodeError:
                collections = []
    client.close()
    return collections


# Function to get documents from a collection
def get_documents_df(collection_name: str):
    collections_dir = os.path.join(SOURCE_FILES_DIR, collection_name)
    # Get all files in the directory, the creation time in date, and the file size in MB
    if not os.path.exists(collections_dir):
        return pd.DataFrame()
    records = []
    for file in os.listdir(collections_dir):
        # get filename and mimetype
        file_name = file
        file_path = os.path.join(collections_dir, file)
        mime_type, _ = mimetypes.guess_type(file_path)

        file_size = os.path.getsize(file_path) / (1024 * 1024)
        file_size = round(file_size, 2)
        file_ctime = os.path.getctime(file_path)
        file_mtime = os.path.getmtime(file_path)
        # convert the creation time in ISO 8601 format
        file_ctime = time.strftime("%Y-%m-%d %H:%M:%S", time.gmtime(file_ctime))
        file_mtime = time.strftime("%Y-%m-%d %H:%M:%S", time.gmtime(file_mtime))
        records.append(
            {
                "File": file_name,
                "Type": mime_type,
                "Size (MB)": file_size,
                "Created At": file_ctime,
                "Modified At": file_mtime,
            }
        )
    # Create a DataFrame from the records
    records_df = pd.DataFrame(records)
    return records_df.drop_duplicates().reset_index(drop=True)


# Function to get the table name from a data source ID
def table_name_from(data_source_id: int):
    return f"index_{data_source_id}"


# Function to get or create a Qdrant vector store
def get_or_create_qdrant_vector_store(
    data_source_id: str, vector_size: int, distance_metric: str
):
    client = QdrantClient(host="localhost", port=6333)
    vector_store = QdrantVectorStore(
        table_name_from(data_source_id),
        client,
        dense_config=VectorParams(size=vector_size, distance=distance_metric),
    )
    return vector_store


# Function to create a new collection
def create_collection(
    name: str,
    vector_size: int,
    distance_metric: str,
    chunk_size: int,
    chunk_overlap: int,
):
    """
    Creates a collection with the specified name, vector size, and distance metric.
    Parameters:
    name (str): The name of the collection to be created.
    vector_size (int): The size of the vectors in the collection.
    distance_metric (str): The distance metric to be used for the collection.
                           Must be one of "Cosine", "Euclidean", or "Dot".
    Raises:
    ValueError: If an invalid distance metric is provided.
    """
    if distance_metric == "Cosine":
        distance_metric = Distance.COSINE
    elif distance_metric == "Euclidean":
        distance_metric = Distance.EUCLID
    elif distance_metric == "Dot":
        distance_metric = Distance.DOT
    else:
        raise ValueError(f"Invalid distance metric: {distance_metric}")

    # Save the collection configuration to a JSON file
    collections = get_collections()
    collection_config = {
        "id": len(collections) + 1,
        "name": name,
        "vector_size": vector_size,
        "distance_metric": distance_metric,
        "chunk_size": chunk_size,
        "chunk_overlap": chunk_overlap,
    }
    collections.append(collection_config)
    with open(COLLECTIONS_JSON, "w+") as f:
        json.dump(collections, f)

    client = QdrantClient(url="http://localhost:6333")
    client.create_collection(
        collection_name=table_name_from(collection_config["id"]),
        vectors_config=VectorParams(size=vector_size, distance=distance_metric),
    )


# Function to save an uploaded file
def save_uploadedfile(uploadedfile, collection_name):
    """
    takes the temporary file from streamlit upload and saves it
    """
    save_dir = os.path.join(SOURCE_FILES_DIR, collection_name)
    if not os.path.exists(save_dir):
        os.makedirs(save_dir, exist_ok=True)

    save_path = os.path.join(save_dir, uploadedfile.name)
    try:
        with open(save_path, "wb") as f:
            f.write(uploadedfile.getbuffer())
        return save_path
    except Exception as e:
        st.error(f"Error saving file {uploadedfile.name}: {e}")
        return None


# Function to upload and index documents to a collection
def upload_documents(collection_config: RagIndexConfiguration):
    """
    Upload documents to the specified collection.
    Parameters:
    collection_config (RagIndexConfiguration): The configuration of the collection to upload documents to.
    """
    uploaded_files = st.file_uploader(
        "Choose files to upload",
        accept_multiple_files=True,
        key=f"upload_files_{collection_config.id}",
    )
    st.info("Click the button below to index documents after uploading", icon="ℹ️")
    if st.button("Add Documents", key=f"upload_doc_to_index_{collection_config.id}"):
        if uploaded_files:
            for uploaded_file in uploaded_files:
                save_uploadedfile(uploaded_file, table_name_from(collection_config.id))
            tmpdir = os.path.join(
                SOURCE_FILES_DIR, table_name_from(collection_config.id)
            )
            distance_metric = collection_config.distance_metric
            if distance_metric == "Cosine":
                distance_metric = Distance.COSINE
            elif distance_metric == "Euclidean":
                distance_metric = Distance.EUCLID
            elif distance_metric == "Dot":
                distance_metric = Distance.DOT
            else:
                raise ValueError(f"Invalid distance metric: {distance_metric}")
            vector_store = get_or_create_qdrant_vector_store(
                data_source_id=collection_config.id,
                vector_size=collection_config.vector_size,
                distance_metric=distance_metric,
            )
            storage_context = StorageContext.from_defaults(vector_store=vector_store)

            chunk_overlap_tokens = int(
                collection_config.chunk_overlap * 0.01 * collection_config.chunk_size
            )
            documents = SimpleDirectoryReader(tmpdir).load_data()
            VectorStoreIndex.from_documents(
                documents,
                storage_context=storage_context,
                show_progress=False,
                transformations=[
                    SentenceSplitter(
                        chunk_size=collection_config.chunk_size,
                        chunk_overlap=chunk_overlap_tokens,
                    )
                ],
            )
            st.success("Documents added successfully!")
        else:
            st.warning("No files added for upload.")


# modal to create a new collection
@st.dialog("Create Data Source")
def create_collection_modal():
    name = st.text_input("Name")
    vector_size = st.number_input("Vector Size", value=EMBED_DIMS, disabled=True)
    distance_metric = st.selectbox(
        "Distance Metric", ["Cosine", "Euclidean", "Dot"], disabled=True
    )
    chunk_size = st.number_input("Chunk Size", value=512)
    chunk_overlap = st.number_input("Chunk Overlap (%)", value=0)

    if st.button("Create", key="create_collection"):
        create_collection(name, vector_size, distance_metric, chunk_size, chunk_overlap)
        st.success(f"Data Source '{name}' created successfully!")
        st.rerun()


# Streamlit page to manage data sources

st.title(
    ":material/stacks: Data Sources", help="Manage data sources for the RAG index."
)

_, create_col, _ = st.columns([4, 1, 4])

with create_col:
    st.image(os.path.join(RESOURCES_DIR, "database.png"), use_column_width=True)

_, create_button_col, _ = st.columns([11, 3, 11])

with create_button_col:
    if st.button("Create New Collection", key="create_collection_modal"):
        create_collection_modal()

# Display list of collections
collections = get_collections()
collections_placeholder = st.empty()
with collections_placeholder.container():
    if collections:
        for collection in collections:
            collection_config = RagIndexConfiguration(**collection)
            with st.expander(f"**:material/database: {collection['name']}**"):
                idcol, sizecol, distcol = st.columns([1, 1, 1])
                idcol.write("**ID**")
                idcol.caption(f"{collection['id']}")
                sizecol.write("**Vector Size**")
                sizecol.caption(f"{collection['vector_size']}")
                distcol.write("**Distance Metric**")
                distcol.caption(f"{collection['distance_metric']}")
                upload_documents(collection_config=collection_config)
                documents_df = get_documents_df(
                    collection_name=table_name_from(collection["id"])
                )
                if not documents_df.empty:
                    st.write(":material/lists: **Documents indexed**")
                    st.dataframe(documents_df)
    else:
        st.write("No collections found.")
