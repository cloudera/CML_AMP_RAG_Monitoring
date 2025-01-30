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

import logging
import os
import sys
import threading
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from uvicorn.logging import DefaultFormatter

from .config import settings
from .routers import index
from .utils.reconciler import background_worker
from .utils.evaluate import evaluate_json_data

###################################
#  Logging
###################################

logger = logging.getLogger(__name__)


def _configure_logger() -> None:
    """Configure this module's setup/teardown logging formatting and verbosity."""
    # match uvicorn.error's formatting
    formatter = DefaultFormatter("%(levelprefix)s %(message)s")

    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(formatter)

    logger.addHandler(handler)
    logger.setLevel(settings.rag_log_level)
    # prevent duplicate outputs with the app logger
    logger.propagate = False


_configure_logger()

###################################
#  Initialization of directories
###################################


def initialize_directories():
    """Initialize directories."""
    data_directory = os.path.join(os.getcwd(), "data")
    if not os.path.exists(data_directory):
        logger.info("Data directory doesn't exist. Creating data directory.")
        os.makedirs(data_directory)
    logger.info("Data directory: %s", data_directory)
    response_directory = os.path.join(data_directory, "responses")
    if not os.path.exists(response_directory):
        logger.info("Responses directory doesn't exist. Creating responses directory.")
        os.makedirs(os.path.join(data_directory, "responses"))
    logger.info("Responses directory: %s", response_directory)
    indexed_files_directory = os.path.join(data_directory, "indexed_files")
    if not os.path.exists(indexed_files_directory):
        logger.info(
            "Indexed files directory doesn't exist. Creating indexed files directory."
        )
        os.makedirs(indexed_files_directory)
    logger.info("Indexed files directory: %s", indexed_files_directory)
    collections_directory = os.path.join(data_directory, "collections")
    if not os.path.exists(collections_directory):
        logger.info(
            "Collections directory doesn't exist. Creating collections directory."
        )
        os.makedirs(collections_directory)
    logger.info("Collections directory: %s", collections_directory)
    custom_evaluators_directory = os.path.join(data_directory, "custom_evaluators")
    if not os.path.exists(custom_evaluators_directory):
        logger.info(
            "Custom evaluators directory doesn't exist. Creating custom evaluators directory."
        )
        os.makedirs(custom_evaluators_directory)
    logger.info("Custom evaluators directory: %s", custom_evaluators_directory)
    logger.info("Directories initialized.")


initialize_directories()


###################################
# Reconciler
###################################


def start_evaluation_reconciler():
    """Start the evaluation reconciler."""
    logger.info("Starting evaluation reconciler.")
    responses_directory = os.path.join(os.getcwd(), "data", "responses")
    if not os.path.exists(responses_directory):
        logger.info("Data directory doesn't exist. Creating data directory.")
        os.makedirs(responses_directory)
    logger.info("Reconciler looking for files in directory: %s", responses_directory)
    worker_thread = threading.Thread(
        target=background_worker,
        args=(responses_directory, evaluate_json_data),
        daemon=True,
    )
    worker_thread.start()
    logger.info("Evaluation reconciler started.")


@asynccontextmanager
async def lifespan(fastapi_app: FastAPI):
    """Initialize and teardown the application's lifespan events."""
    start_evaluation_reconciler()
    yield


###################################
#  App
###################################

# TODO: add reconciler to app lifecycle
app = FastAPI(lifespan=lifespan)


@app.get("/health")
async def health_check():
    return {"status": "ok"}


###################################
#  Middleware
###################################

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

###################################
#  Routes
###################################

app.include_router(index.router)
