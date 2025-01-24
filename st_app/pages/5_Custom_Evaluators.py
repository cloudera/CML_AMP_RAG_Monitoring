import os
import json
import requests
from pathlib import Path
import streamlit as st

from data_types import CreateCustomEvaluatorRequest

custom_evals_dir = Path(os.path.join(os.getcwd(), "custom_evaluators"))


def get_custom_evaluators():
    custom_evaluators = []
    if not custom_evals_dir.exists():
        return custom_evaluators

    for file in os.listdir(custom_evals_dir):
        if file.endswith(".json"):
            # read the json file
            with open(os.path.join(custom_evals_dir, file), "r") as f:
                custom_evaluators.append(json.load(f))
    return custom_evaluators


def add_custom_evaluator(request: CreateCustomEvaluatorRequest):
    fastapi_port = os.environ["FASTAPI_PORT"]
    if fastapi_port is None:
        fastapi_port = 8000

    response = requests.post(
        url=f"http://localhost:{fastapi_port}/index/add_custom_evaluator",
        data=request.json(),
        headers={
            "Content-Type": "application/json",
            "Accept": "application/json",
        },
        timeout=60,
    )

    if response.json().get("status") != "success":
        return False

    return True


@st.dialog("Create Custom Evaluator")
def create_evaluator_modal():
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
        request = CreateCustomEvaluatorRequest(
            name=name,
            eval_definition=evaluator_definition,
            questions=questions,
        )
        if add_custom_evaluator(request):
            st.success("Custom Evaluator Created")
            st.rerun()
        else:
            st.error("Failed to create custom evaluator")


st.title("Custom Evaluators")
st.markdown(
    """
    Custom evaluators are used to evaluate the quality of the generated responses. 
    You can create custom evaluators by defining the evaluator and a set of questions.
    """
)

custom_evaluators = get_custom_evaluators()

custom_evaluators_placeholder = st.empty()
with custom_evaluators_placeholder:
    if custom_evaluators:
        for evaluator in custom_evaluators:
            evaluator_json = CreateCustomEvaluatorRequest(**evaluator)
            with st.expander(f"**:material/function: {evaluator_json.name}**"):
                st.write("**Definition**")
                st.caption(evaluator_json.eval_definition)
                st.write("**Questions**")
                questions = evaluator_json.questions.split("\n")
                for question in questions:
                    st.caption(f"{question}")
    else:
        st.write("No custom evaluators found")

if st.button("Create Custom Evaluator", key="create_evaluator"):
    create_evaluator_modal()
