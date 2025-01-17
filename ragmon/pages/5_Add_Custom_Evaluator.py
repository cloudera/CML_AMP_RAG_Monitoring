import streamlit as st

st.markdown(
    """
    # Add Custom Evaluator
    """
)

st.write(
    """
    This page allows you to add a custom evaluator to the system. 
    """
)

st.write(
    """
    To add a custom evaluator, please fill out the form below.
    """
)

st.write(
    """
    ## Custom Evaluator Form
    """
)

evaluator_name = st.text_input("Evaluator Name")
evaluator_definition = st.text_area("Evaluator Definition")
evaluator_questions = st.text_area("Evaluator Questions")

# add a sub form for evaluation examples
st.write(
    """
    ## Evaluation Examples
    """
)

example_input = st.text_area("Input")
example_evaluation = st.text_area("Evaluation")
