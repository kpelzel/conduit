# Copyright 2026. Triad National Security, LLC. All rights reserved.

from __future__ import annotations
from dataclasses import dataclass, field
import os
from typing import Optional


def _env_str(name: str, default: Optional[str] = None) -> Optional[str]:
    v = os.getenv(name)
    return v if v is not None else default


def _default_addr() -> str:
    return _env_str("CONDUIT_CLI_CONDUIT_IP", "")


def _default_ca_path() -> Optional[str]:
    return _env_str("CONDUIT_CLI_CONDUIT_CA", "")


def _default_cert_key_bundle() -> Optional[str]:
    return _env_str("CONDUIT_CLI_CERT_KEY_BUNDLE", None)


@dataclass(frozen=True)
class ConduitClientConfig:
    """
    Configuration used to connect to the gRPC server.
    """

    addr: str = field(default_factory=_default_addr)
    timeout_s: float = 10.0
    grpc_limit: int = (
        100000000  # The size limit (in bytes) of grpc messages received from conduit
    )

    # TLS/mTLS
    ca_pem_path: Optional[str] = field(default_factory=_default_ca_path)
    cert_key_bundle_path: Optional[str] = field(
        default_factory=_default_cert_key_bundle
    )
    cert_pem_path: Optional[str] = None  # separate client cert (if not combined)
    key_pem_path: Optional[str] = None  # separate client key (if not combined)
