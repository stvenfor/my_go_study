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


def node_title_for_path(relative_path: str) -> str:
    name = Path(relative_path).name
    if name.lower() == "readme.md":
        return "README"
    return Path(relative_path).stem


def directory_segments(relative_path: str) -> list[str]:
    parts = Path(relative_path).parts
    if len(parts) <= 1:
        return []
    return list(parts[:-1])
