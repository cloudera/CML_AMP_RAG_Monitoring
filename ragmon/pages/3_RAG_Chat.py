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

import os
import time
from pathlib import Path
import streamlit as st
from qdrant_client import QdrantClient
import uuid
import requests
import json
from data_types import (
    RagPredictConfiguration,
    RagPredictRequest,
    RagPredictResponse,
    RagMessage,
    RagFeedbackRequest,
)

# get collections directory
file_path = Path(os.path.realpath(__file__))
st_app_dir = file_path.parents[1]
COLLECTIONS_JSON = os.path.join(st_app_dir, "collections.json")


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


# Function to get response from the backend
def get_response(request: RagPredictRequest) -> RagPredictResponse:
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


# Function to show feedback component
def show_feedback_component():
    """feedback component"""
    if st.session_state.responses:
        last_response = st.session_state.responses[-1]
        fb_button = st.feedback(key="feedback_{last_response.id}")
        if fb_button == 1:
            feedback_request = RagFeedbackRequest(
                experiment_id=last_response.mlflow_experiment_id,
                experiment_run_id=last_response.mlflow_run_id,
                feedback=1.0,
            )
            log_feedback(feedback_request)
            st.success("Thanks! :material/sentiment_satisfied:")
            st.session_state.feedback_store[-1] = True
            time.sleep(2)
            st.rerun()
        elif fb_button == 0:
            feedback_request = RagFeedbackRequest(
                experiment_id=last_response.mlflow_experiment_id,
                experiment_run_id=last_response.mlflow_run_id,
                feedback=0.0,
            )
            log_feedback(feedback_request)
            st.error("sorry :material/sentiment_dissatisfied:")
            st.session_state.feedback_store[-1] = True
            time.sleep(2)
            st.rerun()


# Function to show text feedback modal
@st.dialog("Can you tell us more about your experience?")
def show_text_feedback_modal():
    """text feedback component"""
    if st.session_state.responses:
        last_response = st.session_state.responses[-1]
        feedback_text = st.text_area(":material/feedback: Please provide feedback")
        thumb = st.radio(
            label="How was your experience?",
            options=[":material/thumb_up:", ":material/thumb_down:"],
        )
        if st.button("Submit"):
            feedback_request = RagFeedbackRequest(
                experiment_id=last_response.mlflow_experiment_id,
                experiment_run_id=last_response.mlflow_run_id,
                feedback=1.0 if thumb == ":material/thumb_up:" else 0.0,
                feedback_str=feedback_text,
            )
            log_feedback(feedback_request)
            st.success(
                "Thanks! " + ":material/sentiment_satisfied:"
                if thumb == ":material/thumb_up:"
                else ":material/sentiment_dissatisfied:"
            )
            st.session_state.feedback_store[-1] = True
            time.sleep(2)
            st.rerun()


# Initialize the chat message history
if "messages" not in st.session_state.keys():
    st.session_state.messages = []

if "session_id" not in st.session_state.keys():
    st.session_state.session_id = uuid.uuid4()

if "responses" not in st.session_state.keys():
    st.session_state.responses = []

if "feedback_store" not in st.session_state.keys():
    st.session_state.feedback_store = []

st.title(":material/forum: RAG chat")
collections = get_collections()

selected_collection = st.selectbox(
    "Choose data source",
    collections,
    index=len(collections) - 1,
    disabled=True if collections == [] else False,
    format_func=lambda x: x["name"],
)

# Display the prior chat messages
for i, message in enumerate(st.session_state.messages):
    with st.chat_message(
        message.role,
        avatar=(
            ":material/person:" if message.role == "user" else ":material/smart_toy:"
        ),
    ):
        st.write(message.content)
        if i == len(st.session_state.messages) - 1 and message.role == "assistant":
            if not st.session_state.feedback_store[-1]:
                _, feedback_col, text_feedback_col = st.columns(
                    [13, 2, 1], vertical_alignment="center"
                )
                with feedback_col:
                    show_feedback_component()
                with text_feedback_col:
                    text_feedback_button = st.button(
                        ":material/feedback:",
                        help="Provide detailed feedback. Coming Soon! :material/sunny:",
                        disabled=True,
                    )
                    if text_feedback_button:
                        show_text_feedback_modal()

prompt_col, reset_col = st.columns([24, 1])

# Chat input
with prompt_col:
    if prompt := st.chat_input(
        "Ask a question!", disabled=True if collections == [] else False
    ):
        st.session_state.messages.append(RagMessage(role="user", content=prompt))
        st.chat_message("user", avatar=":material/person:").markdown(prompt)
        with st.chat_message("assistant", avatar=":material/smart_toy:"):
            with st.spinner("Thinking..."):
                if st.session_state.messages[-1].role != "assistant":
                    chat_history = [
                        RagMessage(role=message.role, content=message.content)
                        for message in st.session_state.messages[:-1]
                    ]
                else:
                    chat_history = []
                request = RagPredictRequest(
                    data_source_id=selected_collection["id"],
                    chat_history=chat_history,
                    query=prompt,
                    configuration=RagPredictConfiguration(),
                )
                response = get_response(request=request)
            # st.markdown(response.output)
            st.session_state.messages = response.chat_history
            st.session_state.responses.append(response)
            st.session_state.feedback_store.append(False)
            st.rerun()

# Reset chat button
with reset_col:
    if st.button(":material/refresh:", help="Reset the chat"):
        st.session_state.messages = []
        st.session_state.responses = []
        st.session_state.session_id = uuid.uuid4()
        st.rerun()
