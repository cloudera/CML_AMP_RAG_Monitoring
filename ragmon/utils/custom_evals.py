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
import json
from typing import Optional, Union, Dict
from pathlib import Path
import streamlit as st
from data_types import CustomEvaluatorRequest


file_path = Path(os.path.realpath(__file__))
st_app_dir = file_path.parents[1]
data_dir = os.path.join(st_app_dir, "data")
custom_evals_dir = Path(os.path.join(data_dir, "custom_evaluators"))


def save_to_disk(
    data,
    directory: Union[str, Path, os.PathLike],
    filename: str,
):
    """
    Helper function to save JSON data to disk.

    Args:
        data: The data to save
        directory (Union[str, Path, os.PathLike]): The directory to save the data
        filename (str): The name of the file to save the data

    Returns:
        None
    """
    with open(os.path.join(directory, filename), "w", encoding="utf-8") as f:
        json.dump(data, f, indent=2)


def get_custom_evals_dir(exp_id: Optional[str] = None):
    """
    Get the custom evaluators directory

    Args:
        exp_id (Optional[str], optional): The experiment ID. Defaults to None.

    Returns:
        Path: The path to the custom evaluators directory for the given experiment ID
    """
    return custom_evals_dir.joinpath(exp_id)


def add_custom_evaluator(
    request: CustomEvaluatorRequest,
) -> Dict[str, str]:
    """Add a custom evaluator"""
    try:
        exp_custom_evals_dir = get_custom_evals_dir(request.mlflow_experiment_id)
        save_to_disk(
            request.dict(),
            exp_custom_evals_dir,
            f"{request.name.lower().replace(' ', '_')}.json",
        )
        return True
    except Exception as e:
        return False


def get_custom_evaluators(exp_id: Optional[str] = None):
    custom_evaluators = []
    exp_custom_evals_dir = get_custom_evals_dir(exp_id)
    if not exp_custom_evals_dir.exists():
        return custom_evaluators
    for file in os.listdir(exp_custom_evals_dir):
        if file.endswith(".json"):
            # read the json file
            with open(os.path.join(exp_custom_evals_dir, file), "r") as f:
                custom_evaluators.append(json.load(f))
    return custom_evaluators


@st.dialog("Create Custom Evaluator")
def create_evaluator_modal(experiment_id: Optional[str] = None):
    """
    Create a modal to create a custom evaluator

    Args:
        experiment_id (Optional[str], optional): The experiment ID. Defaults to None.

    Returns:
        None
    """
    name = st.text_input(
        "Evaluator Name",
        help="**Name of the custom evaluator.**  \nFor example:  \n*Friendliness*",
    )
    evaluator_definition = st.text_area(
        "Evaluator Definition",
        help="**Define the evaluator in a sentence or two.**  \nFor example:  \n"
        "*Friendliness assesses the warmth and friendliness of the response.*",
    )
    questions = st.text_area(
        "Questions",
        help="**Newline separated list of questions to use for evaluation.**  \n"
        "For example:  \n *How friendly is the response?*  \n  *How helpful is the response?*",
    )
    if st.button("Create"):
        request = CustomEvaluatorRequest(
            name=name,
            eval_definition=evaluator_definition,
            questions=questions,
            mlflow_experiment_id=experiment_id,
        )
        if add_custom_evaluator(request):
            st.success("Custom Evaluator Created")
            st.rerun()
        else:
            st.error("Failed to create custom evaluator")


def show_custom_evaluators_component(experiment_id: Optional[str] = None):
    """
    Show the custom evaluators tab

    Args:
        experiment_id (Optional[str], optional): The experiment ID. Defaults to None.

    Returns:
        None
    """
    custom_evaluators = get_custom_evaluators(exp_id=experiment_id)
    with st.expander("**:material/functions: Custom Evaluators**", expanded=True):
        st.caption(
            """
            Custom evaluators are used to evaluate the quality of the generated responses. 
            You can create custom evaluators by defining the evaluator and a set of questions.
            """
        )
        if custom_evaluators:
            for evaluator in custom_evaluators:
                evaluator_json = CustomEvaluatorRequest(**evaluator)
                with st.popover(f"**:material/function: {evaluator_json.name}**"):
                    st.write("**Definition**")
                    st.caption(evaluator_json.eval_definition)
                    st.write("**Questions**")
                    questions = evaluator_json.questions.split("\n")
                    for question in questions:
                        st.caption(f"{question}")
        else:
            st.write("No custom evaluators found")
    if st.button("Create Custom Evaluator", key="create_evaluator"):
        create_evaluator_modal(experiment_id=experiment_id)
