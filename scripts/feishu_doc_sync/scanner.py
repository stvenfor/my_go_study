"""Scan markdown files per sync config."""

from __future__ import annotations

import hashlib
from dataclasses import dataclass
from pathlib import Path

import yaml


@dataclass(frozen=True)
class SyncConfig:
    wiki_space_name: str
    wiki_space_description: str
    project_folder: str
    include: list[str]
    exclude: list[str]
    manifest_path: Path
    repo_root: Path
    # Repo-relative prefix stripped before building Feishu directory segments.
    # Example: "docs/grpc" + file docs/grpc/a.md → Feishu path under project folder is just "a".
    strip_prefix: str = ""


@dataclass(frozen=True)
class MarkdownFile:
    relative_path: str
    absolute_path: Path
    content_hash: str


def load_config(config_path: Path) -> SyncConfig:
    with config_path.open(encoding="utf-8") as handle:
        raw = yaml.safe_load(handle)

    wiki = raw.get("wiki", {})
    sync = raw.get("sync", {})
    paths = raw.get("paths", {})
    repo_root = Path(paths.get("repo_root", ".")).resolve()

    return SyncConfig(
        wiki_space_name=wiki.get("space_name", "项目 docs"),
        wiki_space_description=wiki.get(
            "space_description", "各项目 Markdown 文档聚合知识库（Git 自动同步）"
        ),
        project_folder=wiki.get("project_folder", "my_ai_project"),
        include=list(sync.get("include", [])),
        exclude=list(sync.get("exclude", [])),
        manifest_path=(repo_root / paths.get("manifest", "docs/feishu-sync.manifest.json")).resolve(),
        repo_root=repo_root,
        strip_prefix=str(sync.get("strip_prefix", "") or "").strip().strip("/"),
    )


def _matches_any(path: str, patterns: list[str]) -> bool:
    from fnmatch import fnmatch

    normalized = path.replace("\\", "/")
    return any(fnmatch(normalized, pattern) for pattern in patterns)


def scan_markdown_files(config: SyncConfig) -> list[MarkdownFile]:
    files: list[MarkdownFile] = []
    seen: set[str] = set()

    for pattern in config.include:
        for absolute in sorted(config.repo_root.glob(pattern)):
            if not absolute.is_file() or absolute.suffix.lower() != ".md":
                continue
            relative = absolute.relative_to(config.repo_root).as_posix()
            if relative in seen:
                continue
            if _matches_any(relative, config.exclude):
                continue
            seen.add(relative)
            content = absolute.read_bytes()
            digest = hashlib.sha256(content).hexdigest()
            files.append(
                MarkdownFile(
                    relative_path=relative,
                    absolute_path=absolute,
                    content_hash=f"sha256:{digest}",
                )
            )

    return sorted(files, key=lambda item: item.relative_path)


def feishu_relative_path(relative_path: str, strip_prefix: str = "") -> str:
    """Map a repo-relative path to the path used for Feishu folder/title layout."""
    normalized = relative_path.replace("\\", "/").lstrip("/")
    prefix = (strip_prefix or "").strip().strip("/")
    if not prefix:
        return normalized
    if normalized == prefix:
        return Path(normalized).name
    head = prefix + "/"
    if normalized.startswith(head):
        return normalized[len(head) :]
    return normalized


def node_title_for_path(relative_path: str, strip_prefix: str = "") -> str:
    mapped = feishu_relative_path(relative_path, strip_prefix)
    name = Path(mapped).name
    if name.lower() == "readme.md":
        return "README"
    return Path(mapped).stem


def directory_segments(relative_path: str, strip_prefix: str = "") -> list[str]:
    mapped = feishu_relative_path(relative_path, strip_prefix)
    parts = Path(mapped).parts
    if len(parts) <= 1:
        return []
    return list(parts[:-1])
