"""
Pi Sandbox Python SDK

Provides programmatic access to the pi-sandbox runtime API.
Connects to the daemon via Unix socket or HTTP.
"""

from __future__ import annotations

import json
import os
import urllib.parse
from dataclasses import dataclass, field
from typing import Any, Dict, Generator, Iterable, List, Optional, Union


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


@dataclass
class ExecStreamEvent:
    """Event from streaming exec output."""
    event_type: str  # 'stdout', 'stderr', 'done'
    data: str = ""
    exit_code: Optional[int] = None
    duration_ms: Optional[int] = None
    truncated: Optional[bool] = None
    timed_out: Optional[bool] = None


class SandboxClient:
    """Client for the Pi Sandbox API."""

    def __init__(
        self,
        socket_path: Optional[str] = None,
        base_url: Optional[str] = None,
        auth_token: Optional[str] = None,
    ) -> None:
        """Construct a SandboxClient.

        For remote http daemon contexts (F23/ADR-003), pass ``auth_token`` or
        set ``PI_AUTH_TOKEN`` in the environment. The token is never written
        to disk by the SDK.
        """
        self.socket_path = socket_path or os.environ.get(
            "PI_SOCKET_PATH", "~/.pi-box/sandboxd.sock"
        )
        self.base_url = base_url
        self.auth_token = auth_token or os.environ.get("PI_AUTH_TOKEN")

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
        import urllib.error

        url = f"{self.base_url}{path}"
        data = json.dumps(body).encode() if body else None
        req = urllib.request.Request(url, data=data, method=method)
        req.add_header("Content-Type", "application/json")
        if self.auth_token:
            req.add_header("Authorization", f"Bearer {self.auth_token}")

        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode())
        except urllib.error.HTTPError as e:
            if e.code in (401, 403):
                # ADR-003: never fall back to unauthenticated access.
                raise RuntimeError(
                    f"Remote auth failed: HTTP {e.code} {e.reason}. "
                    f"Check the bearer token for this context."
                ) from e
            raise RuntimeError(f"API error: {e.code} {e.reason}")

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

    def destroy_all(self) -> int:
        """Destroy all sandboxes and return count."""
        sandboxes = self.list()
        for sb in sandboxes:
            self.destroy(sb.id)
        return len(sandboxes)

    def exec(
        self,
        sandbox_id: str,
        command: str,
        timeout_ms: Optional[int] = None,
        max_output_bytes: Optional[int] = None,
        cwd: Optional[str] = None,
    ) -> ExecResult:
        result = self._request(
            "POST",
            f"/v1/sandboxes/{sandbox_id}/exec",
            {
                "command": command,
                "timeoutMs": timeout_ms,
                "maxOutputBytes": max_output_bytes,
                "cwd": cwd,
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

    def exec_stream(
        self,
        sandbox_id: str,
        command: str,
        timeout_ms: Optional[int] = None,
        max_output_bytes: Optional[int] = None,
        cwd: Optional[str] = None,
    ) -> Generator[ExecStreamEvent, None, None]:
        """
        Execute a command and stream stdout/stderr events.
        Yields ExecStreamEvent for each chunk of output.
        """
        import urllib.request

        url = f"{self.base_url}/v1/sandboxes/{sandbox_id}/exec?stream=true"
        body = json.dumps({
            "command": command,
            "timeoutMs": timeout_ms,
            "maxOutputBytes": max_output_bytes,
            "cwd": cwd,
        }).encode()

        req = urllib.request.Request(url, data=body, method="POST")
        req.add_header("Content-Type", "application/json")
        req.add_header("Accept", "application/x-ndjson")

        buffer = ""
        try:
            with urllib.request.urlopen(req) as resp:
                while True:
                    chunk = resp.read(4096)
                    if not chunk:
                        break
                    buffer += chunk.decode("utf-8")
                    while "\n" in buffer:
                        line, buffer = buffer.split("\n", 1)
                        if not line.strip():
                            continue
                        try:
                            data = json.loads(line)
                            yield ExecStreamEvent(
                                event_type=data.get("type", "stdout"),
                                data=data.get("data"),
                                exit_code=data.get("exitCode"),
                                duration_ms=data.get("durationMs"),
                                truncated=data.get("truncated"),
                                timed_out=data.get("timedOut"),
                            )
                        except json.JSONDecodeError:
                            yield ExecStreamEvent(event_type="stdout", data=line)
        except Exception as e:
            yield ExecStreamEvent(event_type="done", data=str(e))

    def clone(self, sandbox_id: str, url: str) -> None:
        self._request("POST", f"/v1/sandboxes/{sandbox_id}/clone", {"url": url})

    def diff(self, sandbox_id: str) -> str:
        return self._request("GET", f"/v1/sandboxes/{sandbox_id}/diff")

    def patch(self, sandbox_id: str) -> str:
        return self._request("GET", f"/v1/sandboxes/{sandbox_id}/patch")

    def files_read(self, sandbox_id: str, path: str) -> str:
        return self._request(
            "GET",
            f"/v1/sandboxes/{sandbox_id}/files/read?path={urllib.parse.quote(path)}",
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

    def artifacts_pull(self, sandbox_id: str, destination: str) -> None:
        self._request("POST", f"/v1/sandboxes/{sandbox_id}/artifacts/pull", {"destination": destination})

    def artifacts_pack(self, sandbox_id: str, output: str) -> None:
        self._request("POST", f"/v1/sandboxes/{sandbox_id}/artifacts/pack", {"output": output})

    def snapshot_create(self, sandbox_id: str, name: str) -> None:
        self._request("POST", f"/v1/sandboxes/{sandbox_id}/snapshot/create", {"name": name})

    def snapshot_list(self, sandbox_id: str) -> List[Any]:
        return self._request("GET", f"/v1/sandboxes/{sandbox_id}/snapshot/list")

    def snapshot_rollback(self, sandbox_id: str, name: str) -> None:
        self._request("POST", f"/v1/sandboxes/{sandbox_id}/snapshot/rollback", {"name": name})

    def snapshot_delete(self, sandbox_id: str, name: str) -> None:
        self._request("POST", f"/v1/sandboxes/{sandbox_id}/snapshot/delete", {"name": name})


def create_client(
    socket_path: Optional[str] = None,
    base_url: Optional[str] = None,
    auth_token: Optional[str] = None,
) -> SandboxClient:
    """Create a new SandboxClient."""
    return SandboxClient(socket_path=socket_path, base_url=base_url, auth_token=auth_token)
