"""
Pi Sandbox Python SDK

Provides programmatic access to the pi-sandbox runtime API.
Connects to the daemon via Unix socket or HTTP.
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass, field
from typing import Any, Dict, List, Optional, Union


@dataclass
class ExecResult:
    """Result of an exec command."""
    exit_code: int
    duration_ms: int
    stdout: str
    stderr: str
    truncated: bool
    timed_out: bool


@dataclass
class SandboxInfo:
    """Sandbox metadata."""
    id: str
    name: str
    template: str
    mode: str
    state: str
    created_at: str
    last_used: str


class SandboxClient:
    """Client for the Pi Sandbox API."""

    def __init__(
        self,
        socket_path: Optional[str] = None,
        base_url: Optional[str] = None,
    ) -> None:
        self.socket_path = socket_path or os.environ.get(
            "PI_SOCKET_PATH", "~/.pi/sandboxd.sock"
        )
        self.base_url = base_url

    def _request(
        self,
        method: str,
        path: str,
        body: Optional[Dict[str, Any]] = None,
    ) -> Any:
        if self.base_url:
            return self._http_request(method, path, body)
        raise RuntimeError("SDK requires either socket_path or base_url")

    def _http_request(
        self,
        method: str,
        path: str,
        body: Optional[Dict[str, Any]] = None,
    ) -> Any:
        import urllib.request

        url = f"{self.base_url}{path}"
        data = json.dumps(body).encode() if body else None
        req = urllib.request.Request(url, data=data, method=method)
        req.add_header("Content-Type", "application/json")

        with urllib.request.urlopen(req) as resp:
            return json.loads(resp.read().decode())

    def create(
        self,
        template: str,
        mode: str = "fast",
        name: Optional[str] = None,
    ) -> SandboxInfo:
        result = self._request(
            "POST",
            "/v1/sandboxes",
            {"name": name, "template": template, "mode": mode},
        )
        return SandboxInfo(**result)

    def list(self) -> List[SandboxInfo]:
        result = self._request("GET", "/v1/sandboxes")
        return [SandboxInfo(**item) for item in result]

    def get(self, sandbox_id: str) -> SandboxInfo:
        result = self._request("GET", f"/v1/sandboxes/{sandbox_id}")
        return SandboxInfo(**result)

    def destroy(self, sandbox_id: str) -> None:
        self._request("DELETE", f"/v1/sandboxes/{sandbox_id}")

    def exec(
        self,
        sandbox_id: str,
        command: str,
        timeout_ms: Optional[int] = None,
        max_output_bytes: Optional[int] = None,
    ) -> ExecResult:
        result = self._request(
            "POST",
            f"/v1/sandboxes/{sandbox_id}/exec",
            {
                "command": command,
                "timeoutMs": timeout_ms,
                "maxOutputBytes": max_output_bytes,
            },
        )
        return ExecResult(
            exit_code=result["exitCode"],
            duration_ms=result["durationMs"],
            stdout=result["stdout"],
            stderr=result["stderr"],
            truncated=result["truncated"],
            timed_out=result["timedOut"],
        )

    def clone(self, sandbox_id: str, url: str) -> None:
        self._request("POST", f"/v1/sandboxes/{sandbox_id}/clone", {"url": url})

    def diff(self, sandbox_id: str) -> str:
        return self._request("GET", f"/v1/sandboxes/{sandbox_id}/diff")

    def patch(self, sandbox_id: str) -> str:
        return self._request("GET", f"/v1/sandboxes/{sandbox_id}/patch")

    def files_read(self, sandbox_id: str, path: str) -> str:
        return self._request(
            "GET",
            f"/v1/sandboxes/{sandbox_id}/files/read",
            {"path": path},
        )

    def files_write(self, sandbox_id: str, path: str, content: str) -> None:
        self._request(
            "POST",
            f"/v1/sandboxes/{sandbox_id}/files/write",
            {"path": path, "content": content},
        )

    def logs(self, sandbox_id: str) -> List[ExecResult]:
        result = self._request("GET", f"/v1/sandboxes/{sandbox_id}/logs")
        return [
            ExecResult(
                exit_code=item["exitCode"],
                duration_ms=item["durationMs"],
                stdout=item["stdout"],
                stderr=item["stderr"],
                truncated=item["truncated"],
                timed_out=item["timedOut"],
            )
            for item in result
        ]

    def artifacts_list(self, sandbox_id: str) -> List[Any]:
        return self._request("GET", f"/v1/sandboxes/{sandbox_id}/artifacts")


def create_client(
    socket_path: Optional[str] = None,
    base_url: Optional[str] = None,
) -> SandboxClient:
    """Create a new SandboxClient."""
    return SandboxClient(socket_path=socket_path, base_url=base_url)
