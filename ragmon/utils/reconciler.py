"""
Reconciler script to process io request pairs.
"""

import os
import json
import logging
import time
import threading
from pathlib import Path
import argparse
from typing import Union
import asyncio
from typing import Callable

from .evaluate import evaluate_json_data
from ..config import settings

# Configure logging
logger = logging.getLogger(__name__)


async def process_io_pair(file_path, processing_function):
    """Callback function to process a io saved in a file."""
    with open(file_path, "r", encoding="utf-8") as f:
        data = json.load(f)
    logger.info("Processing i/o pair: %s", file_path)
    # Process io pair
    await processing_function(data)
    logger.info("Processed i/o pair: %s", file_path)


def background_worker(directory, processing_function):
    """Background thread function to process files."""
    if not isinstance(directory, Path):
        directory = Path(directory)
    loop = asyncio.new_event_loop()
    asyncio.set_event_loop(loop)
    while True:
        for file_path in directory.iterdir():
            if file_path.is_file() and file_path.suffix == ".json":
                try:
                    loop.run_until_complete(
                        process_io_pair(
                            file_path=file_path, processing_function=processing_function
                        )
                    )
                    file_path.unlink()  # Delete the file after processing
                except Exception as e:
                    logger.error("Error processing file %s: %s", file_path, e)
        time.sleep(1)


# Start background worker thread
def start_background_worker(directory, processing_function):
    """Start the background worker thread."""
    if not isinstance(directory, Path):
        directory = Path(directory)
    worker_thread = threading.Thread(
        target=background_worker, args=(directory, processing_function), daemon=True
    )
    worker_thread.start()


if __name__ == "__main__":
    # Directory to save JSON files
    # Argument parsing to get the data directory
    parser = argparse.ArgumentParser(
        description="Reconciler script to process io request pairs."
    )
    parser.add_argument(
        "--data-dir", type=str, default="data", help="Directory to save JSON files"
    )
    args = parser.parse_args()

    data_dir = Path(args.data_dir)
    data_dir.mkdir(exist_ok=True)
    start_background_worker(data_dir, evaluate_json_data)
    try:
        while True:
            logger.info("Reconciler looking for i/o pairs in %s...", data_dir)
            time.sleep(1)
    except KeyboardInterrupt:
        logger.info("Shutting down...")
