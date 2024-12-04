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
import streamlit as st
from pathlib import Path

# get resources directory
file_path = Path(os.path.realpath(__file__))
logo_path = os.path.join(
    file_path.parents[0], "resources", "logos", "RAG-Monitoring-icon.png"
)

# Setup the navigation
def setup_navigation():
    pg = st.navigation(
            [
                st.Page("pages/1_Home.py", title="Home"),
                st.Page(
                    "pages/2_Data_Source.py",
                    title="Data Sources",
                ),
                st.Page(
                    "pages/3_RAG_Chat.py",
                    title="RAG Chat",
                ),
                st.Page(
                    "pages/4_Monitoring_Dashboard.py",
                    title="Monitoring Dashboard",
                ),
                st.Page(
                    "pages/5_Leave_Feedback.py",
                    title="Leave Feedback",
                ),
            ],
            position="hidden",
        )
    centered_pages = ["Home"]
    st.set_page_config(
        layout="centered" if pg.title in centered_pages else "wide",
        page_title="RAG Monitoring AMP",
    )
    pg.run()

# Setup the sidebar
def setup_sidebar():
    with st.sidebar:
        st.image(logo_path, use_column_width=True)
        st.markdown(
            """
            :orange-background[:material/wb_sunny: **Technical Preview** ]
            """
        )
        st.page_link("pages/1_Home.py", label="Home", icon=":material/home:")
        st.page_link(
            "pages/2_Data_Source.py",
            label="Data Sources",
            icon=":material/stacks:",
        )
        st.page_link("pages/3_RAG_Chat.py", label="RAG Chat", icon=":material/forum:")
        st.page_link(
            "pages/4_Monitoring_Dashboard.py",
            label="Monitoring Dashboard",
            icon=":material/monitoring:",
        )
        st.page_link(
            "pages/5_Leave_Feedback.py",
            label="Leave Feedback",
            icon=":material/comment:",
        )

# Main function to orchestrate the setup
setup_navigation()
setup_sidebar()

