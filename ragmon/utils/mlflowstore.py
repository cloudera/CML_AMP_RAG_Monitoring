import http
import logging
import sys
import requests
from uvicorn.logging import DefaultFormatter

from ..data_types import MLflowStoreIdentifier
from ..config import settings

logger = logging.getLogger(__name__)
formatter = DefaultFormatter("%(levelprefix)s %(message)s")

handler = logging.StreamHandler(sys.stdout)
handler.setFormatter(formatter)

logger.addHandler(handler)
logger.setLevel(settings.rag_log_level)


def register_experiment_and_run(
    experiment_id: str,
    experiment_run_id: str,
) -> bool:
    try:
        mlflowstore = MLflowStoreIdentifier(
            experiment_id=experiment_id,
            experiment_run_id=experiment_run_id,
        )
        response = requests.post(
            url=f"{settings.mlflow_store.uri}/runs",
            data=mlflowstore.json(),
            headers={"Content-Type": "application/json"},
            timeout=10,
        )
        if response.status_code != http.HTTPStatus.OK:
            logger.error(
                "Failed to register experiment and run with MLflow store: %s",
                response.text,
            )
            return False
        logger.info(
            "Registered experiment id %s and run id %s with MLflow store",
            experiment_id,
            experiment_run_id,
        )
        return True
    except Exception as e:
        logger.error("Failed to register experiment and run with MLflow store: %s", e)
        return False
