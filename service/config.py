"""
RAG backend configuration.

All configuration values can be set as environment variables; the variable name is
simply the field name in all capital letters.

"""

import logging
from typing import Optional

from pydantic_settings import BaseSettings, SettingsConfigDict


class OTelSettings(BaseSettings, str_strip_whitespace=True):
    """
    OpenTelemetry configuration.

    Environment variable names are taken from the OTel spec. This exists to enforce
    values we require and set our own defaults; otherwise OTel will use its defaults
    which could error at runtime.

    """

    model_config = SettingsConfigDict(env_prefix="otel_")

    service_name: str = "llm-service"
    exporter_otlp_endpoint: Optional[str] = None


class MLFlowSettings(BaseSettings, str_strip_whitespace=True):
    """MLFlow configuration."""

    tracking_uri: str = "http://localhost:5000"


class MLFlowStoreSettings(BaseSettings, str_strip_whitespace=True):
    """MLFlow configuration."""

    uri: str = "http://localhost:3000"


class Settings(BaseSettings):
    """RAG configuration."""

    otel: OTelSettings = OTelSettings()
    mlflow: MLFlowSettings = MLFlowSettings()
    mlflow_store: MLFlowStoreSettings = MLFlowStoreSettings()

    rag_log_level: int = logging.INFO


settings = Settings()
