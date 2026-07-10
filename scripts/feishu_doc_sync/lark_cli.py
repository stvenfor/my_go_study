"""Thin wrapper around the official lark-cli binary."""

from __future__ import annotations

import json
import shutil
import subprocess
from pathlib import Path
from typing import Any

_working_directory: Path | None = None


class LarkCliError(RuntimeError):
    pass


def set_working_directory(path: Path) -> None:
    global _working_directory
    _working_directory = path.resolve()


def _file_content_arg(file_path: str) -> str:
    if _working_directory is None:
        raise LarkCliError("Call set_working_directory(repo_root) before syncing files.")
    absolute = Path(file_path).resolve()
    relative = absolute.relative_to(_working_directory)
    return f"@{relative.as_posix()}"


def ensure_lark_cli() -> str:
    path = shutil.which("lark-cli")
    if not path:
        raise LarkCliError(
            "lark-cli not found. Install: https://github.com/larksuite/cli#installation"
        )
    return path


def run_lark_cli(
    *args: str,
    as_identity: str = "user",
    jq: str | None = None,
) -> Any:
    ensure_lark_cli()
    command = ["lark-cli", *args, "--as", as_identity]
    if jq:
        command.extend(["--jq", jq])
    else:
        command.append("--json")

    result = subprocess.run(
        command,
        capture_output=True,
        text=True,
        check=False,
        cwd=str(_working_directory) if _working_directory else None,
    )
    if result.returncode != 0:
        detail = result.stderr.strip() or result.stdout.strip() or "unknown error"
        raise LarkCliError(f"lark-cli failed ({result.returncode}): {' '.join(args)}\n{detail}")

    stdout = result.stdout.strip()
    if not stdout:
        return None
    if jq:
        try:
            return json.loads(stdout)
        except json.JSONDecodeError:
            return stdout.strip('"')
    start = stdout.find("{")
    if start < 0:
        raise LarkCliError(f"Expected JSON from lark-cli, got: {stdout[:200]}")
    payload = json.loads(stdout[start:])
    if isinstance(payload, dict) and payload.get("ok") is False:
        raise LarkCliError(f"lark-cli error: {payload.get('message', payload)}")
    return payload


def unwrap_data(payload: Any) -> dict[str, Any]:
    if isinstance(payload, dict) and "data" in payload:
        data = payload["data"]
        return data if isinstance(data, dict) else {"value": data}
    return payload if isinstance(payload, dict) else {"value": payload}


def list_wiki_spaces(as_identity: str = "user") -> list[dict[str, Any]]:
    payload = run_lark_cli("wiki", "+space-list", as_identity=as_identity, jq=".data.spaces")
    if isinstance(payload, list):
        return payload
    return []


def find_wiki_space_by_name(name: str, as_identity: str = "user") -> dict[str, Any] | None:
    for space in list_wiki_spaces(as_identity=as_identity):
        if space.get("name") == name:
            return space
    return None


def create_wiki_space(
    name: str,
    description: str,
    as_identity: str = "user",
) -> dict[str, Any]:
    payload = run_lark_cli(
        "wiki",
        "+space-create",
        "--name",
        name,
        "--description",
        description,
        as_identity=as_identity,
        jq=".data",
    )
    return payload if isinstance(payload, dict) else {}


def list_wiki_nodes(
    space_id: str,
    *,
    parent_node_token: str | None = None,
    as_identity: str = "user",
) -> list[dict[str, Any]]:
    args = ["wiki", "+node-list", "--space-id", space_id, "--page-all"]
    if parent_node_token:
        args.extend(["--parent-node-token", parent_node_token])
    payload = run_lark_cli(*args, as_identity=as_identity, jq=".data.items")
    if isinstance(payload, list):
        return payload
    return []


def find_wiki_node_by_title(
    space_id: str,
    title: str,
    *,
    parent_node_token: str | None = None,
    as_identity: str = "user",
) -> dict[str, Any] | None:
    for node in list_wiki_nodes(
        space_id,
        parent_node_token=parent_node_token,
        as_identity=as_identity,
    ):
        if node.get("title") == title:
            return node
    return None


def create_wiki_node(
    space_id: str,
    title: str,
    *,
    parent_node_token: str | None = None,
    as_identity: str = "user",
) -> dict[str, Any]:
    args = ["wiki", "+node-create", "--space-id", space_id, "--title", title, "--obj-type", "docx"]
    if parent_node_token:
        args.extend(["--parent-node-token", parent_node_token])
    payload = run_lark_cli(*args, as_identity=as_identity, jq=".data")
    return payload if isinstance(payload, dict) else {}


def create_document_from_markdown(
    file_path: str,
    title: str,
    *,
    parent_token: str | None = None,
    as_identity: str = "user",
) -> dict[str, Any]:
    args = [
        "docs",
        "+create",
        "--title",
        title,
        "--doc-format",
        "markdown",
        "--content",
        _file_content_arg(file_path),
    ]
    if parent_token:
        args.extend(["--parent-token", parent_token])
    payload = run_lark_cli(*args, as_identity=as_identity, jq=".data.document")
    return payload if isinstance(payload, dict) else {}


def overwrite_document_markdown(
    document_id: str,
    file_path: str,
    as_identity: str = "user",
) -> None:
    run_lark_cli(
        "docs",
        "+update",
        "--doc",
        document_id,
        "--command",
        "overwrite",
        "--doc-format",
        "markdown",
        "--content",
        _file_content_arg(file_path),
        as_identity=as_identity,
    )
