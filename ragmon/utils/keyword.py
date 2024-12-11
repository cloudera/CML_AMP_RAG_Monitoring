from keybert import KeyBERT
from llama_index.llms.bedrock_converse import BedrockConverse
from typing import List

from ragmon.services.ragllm import get_inference_model

EXTRACT_KEYWORDS_PROMPT = """I have the following document:
{input}

Based on the information above, extract the keywords that best describe the topic of the text.
Make sure to only extract keywords that appear in the text. Only return the top {top_n} keywords from the text.
Use the following format separated by commas:
<keywords>

Keywords:
"""

FILTER_CANDIDATES_PROMPT = """
I have the following document:
{input}

With the following candidate keywords:
{candidates}

Based on the information above, improve the candidate keywords to best describe the topic of the document.
Only return the keywords. Do not include any other information. Return at most {top_n} keywords.
Use the following format separated by commas:
<keywords>

Example:
Input: Both Custom and Cloudera provided Runtimes can be added to CML workspaces. Cloudera provided Runtime Repo files contain the details of the latest released ML Runtimes and Data Visualization Runtimes.
Candidates: Custom, Cloudera, Runtimes, CML workspaces, Cloudera provided Runtime Repo files, ML Runtimes, Data Visualization Runtimes
Keywords: Runtimes, CML workspaces, ML Runtimes, Data Visualization Runtimes

Keywords:
"""


def extract_keywords(
    text,
    top_n=10,
    use_llm=False,
    use_keybert=True,
    keybert_model="all-MiniLM-L6-v2",
) -> List[str]:
    """
    Extract keywords from a given text using the specified models.

    Args:
        text: The text to extract keywords from.
        use_llm: Whether to use the LLM model to extract keywords.
        use_keybert: Whether to use the KeyBERT model to extract keywords.
        language: The language of the text.
        keybert_model: The KeyBERT model id to use, should be a huggingface embeddings model. Defaults to "all-MiniLM-L6-v2".

    Returns:
        A list of extracted keywords.
    """
    if use_llm and not use_keybert:
        llm = get_inference_model()
        response = llm.complete(EXTRACT_KEYWORDS_PROMPT.format(input=text, top_n=top_n))
        keywords = [x.strip() for x in response.text.strip().split(",")]
    elif use_keybert and not use_llm:
        model = KeyBERT(model=keybert_model)
        keywords = model.extract_keywords(text)
        keywords = [x[0] for x in keywords]
    elif use_llm and use_keybert:
        llm = get_inference_model()
        model = KeyBERT(model=keybert_model)
        keywords = model.extract_keywords(text, top_n=2 * top_n)
        keywords = [x[0] for x in keywords]
        response = llm.complete(
            FILTER_CANDIDATES_PROMPT.format(
                input=text, candidates=", ".join(keywords), top_n=top_n
            )
        )
        keywords = [x.strip() for x in response.text.strip().split(",")]

    return keywords
