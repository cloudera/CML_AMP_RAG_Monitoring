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
import subprocess

import streamlit as st
import os
from pathlib import Path

# get resources directory
file_path = Path(os.path.realpath(__file__))
logo_path = os.path.join(
    file_path.parents[1], "resources", "logos", "RAG-Monitoring-icon.png"
)


def has_updates():
    result = subprocess.run(
        ["bash /home/cdsw/scripts/check_updates.sh"], shell=True, text=True
    )
    return result.returncode != 0


def restart():
    try:
        import cmlapi
        import os

        client = cmlapi.default_client()
        project_id = os.environ["CDSW_PROJECT_ID"]
        jobs = client.list_jobs(project_id=project_id)
        update_job = None
        for job in jobs.jobs:
            if job.name == "Update/build RAG Monitoring":
                update_job = job
                break
        if update_job is None:
            st.warning("Could not find update job. Please update manually.")
            st.stop()
        client.create_job_run(project_id=project_id, job_id=update_job.id, body={})
    except Exception as e:
        st.warning("Error while fetching job details. Please update manually.")
        st.stop()


@st.dialog("RAG Monitoring is restarting", width="large")
def restarting():
    st.write("You will need to reload the page after the restart.")
    restart()
    st.stop()


if has_updates():
    st.warning(
        "Your RAG Monitoring version is out of date. Please update to the latest version."
    )
    if st.button("Click here to update"):
        restarting()

_, img_col, _ = st.columns([1, 5, 1])
with img_col:
    st.image(logo_path, use_container_width=True)
st.markdown(
    """
Real-time monitoring for AI Studios by best practices and leading frameworks.
"""
)

rag_studio_col, sd_studio_col, ft_studio_col, agent_studio_col = st.columns(
    4, border=True, vertical_alignment="center"
)

with rag_studio_col:
    st.image(
        "https://raw.githubusercontent.com/cloudera/AI-Studios/refs/heads/master/images/rag-studio-banner.svg",
        use_container_width=True,
    )
    st.markdown("**RAG Studio**")
    st.markdown("Real-time monitoring for AI Studios.")
    st.page_link(
        "pages/2_Monitoring_Dashboard.py", label="Open", icon=":material/analytics:"
    )

with sd_studio_col:
    st.image(
        "https://raw.githubusercontent.com/cloudera/AI-Studios/refs/heads/master/images/synthetic-data-studio-banner.svg",
        use_container_width=True,
    )
    st.markdown("**Synthetic Data Studio**")
    st.markdown(":gray-background[:material/wb_sunny: Coming SOON!]")
    st.page_link(
        "pages/2_Monitoring_Dashboard.py",
        label="Open",
        icon=":material/analytics:",
        disabled=True,
    )

with ft_studio_col:
    st.image(
        "https://raw.githubusercontent.com/cloudera/AI-Studios/refs/heads/master/images/fine-tuning-studio-banner.svg",
        use_container_width=True,
    )
    st.markdown("**Fine-tuning Studio**")
    st.markdown(":gray-background[:material/wb_sunny: Coming SOON!]")
    st.page_link(
        "pages/2_Monitoring_Dashboard.py",
        label="Open",
        icon=":material/analytics:",
        disabled=True,
    )

with agent_studio_col:
    st.image(
        "https://raw.githubusercontent.com/cloudera/AI-Studios/refs/heads/master/images/agent-studio-banner.svg",
        use_container_width=True,
    )
    st.markdown("**Agent Studio**")
    st.markdown(":gray-background[:material/wb_sunny: Coming SOON!]")
    st.page_link(
        "pages/2_Monitoring_Dashboard.py",
        label="Open",
        icon=":material/analytics:",
        disabled=True,
    )
