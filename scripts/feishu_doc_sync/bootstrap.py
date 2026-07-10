"""Bootstrap Feishu wiki space and directory tree via lark-cli."""

from __future__ import annotations

import os

import lark_cli
from lark_cli import LarkCliError
from manifest import FileEntry, Manifest
from scanner import MarkdownFile, SyncConfig, directory_segments, node_title_for_path


def _directory_key(segments: list[str]) -> str:
    return "/".join(segments)


def _as_identity() -> str:
    return os.environ.get("FEISHU_SYNC_AS", "user").strip() or "user"


def ensure_wiki_space(config: SyncConfig, manifest: Manifest) -> str:
    if manifest.wiki_space_id:
        return manifest.wiki_space_id

    env_space_id = os.environ.get("FEISHU_WIKI_SPACE_ID", "").strip()
    if env_space_id:
        manifest.wiki_space_id = env_space_id
        manifest.wiki_space_name = config.wiki_space_name
        return env_space_id

    existing = lark_cli.find_wiki_space_by_name(config.wiki_space_name, as_identity=_as_identity())
    if existing:
        space_id = existing.get("space_id", "")
        manifest.wiki_space_id = space_id
        manifest.wiki_space_name = existing.get("name", config.wiki_space_name)
        return space_id

    space = lark_cli.create_wiki_space(
        config.wiki_space_name,
        config.wiki_space_description,
        as_identity=_as_identity(),
    )
    space_id = space.get("space_id", "")
    if not space_id:
        raise LarkCliError("Wiki space creation succeeded but space_id is empty.")
    manifest.wiki_space_id = space_id
    manifest.wiki_space_name = space.get("name", config.wiki_space_name)
    return space_id


def ensure_project_folder(
    space_id: str,
    config: SyncConfig,
    manifest: Manifest,
) -> str:
    if manifest.project_folder_node_token:
        return manifest.project_folder_node_token

    folder_name = config.project_folder
    existing = lark_cli.find_wiki_node_by_title(
        space_id,
        folder_name,
        parent_node_token=None,
        as_identity=_as_identity(),
    )
    if existing:
        token = existing.get("node_token", "")
        manifest.project_folder_name = folder_name
        manifest.project_folder_node_token = token
        return token

    node = lark_cli.create_wiki_node(
        space_id,
        folder_name,
        parent_node_token=None,
        as_identity=_as_identity(),
    )
    token = node.get("node_token", "")
    manifest.project_folder_name = folder_name
    manifest.project_folder_node_token = token
    return token


def ensure_directory_node(
    space_id: str,
    project_root_token: str,
    manifest: Manifest,
    segments: list[str],
) -> str:
    key = _directory_key(segments)
    existing = manifest.get_directory_node(key)
    if existing:
        return existing

    parent_segments = segments[:-1]
    if parent_segments:
        parent_token = ensure_directory_node(space_id, project_root_token, manifest, parent_segments)
    else:
        parent_token = project_root_token

    title = segments[-1]
    node = lark_cli.create_wiki_node(
        space_id,
        title,
        parent_node_token=parent_token,
        as_identity=_as_identity(),
    )
    node_token = node.get("node_token", "")
    manifest.set_directory_node(key, node_token)
    return node_token


def ensure_file_node(
    space_id: str,
    project_root_token: str,
    manifest: Manifest,
    md_file: MarkdownFile,
) -> FileEntry:
    existing = manifest.get_file(md_file.relative_path)
    if existing and existing.node_token and existing.document_id:
        return existing

    segments = directory_segments(md_file.relative_path)
    if segments:
        parent_token = ensure_directory_node(space_id, project_root_token, manifest, segments)
    else:
        parent_token = project_root_token

    title = node_title_for_path(md_file.relative_path)
    node = lark_cli.create_wiki_node(
        space_id,
        title,
        parent_node_token=parent_token,
        as_identity=_as_identity(),
    )
    entry = FileEntry(
        node_token=node.get("node_token", ""),
        document_id=node.get("obj_token", ""),
        content_hash="",
    )
    manifest.set_file(md_file.relative_path, entry)
    return entry


def sync_markdown_to_document(document_id: str, md_file: MarkdownFile) -> None:
    lark_cli.overwrite_document_markdown(
        document_id,
        str(md_file.absolute_path),
        as_identity=_as_identity(),
    )


def bootstrap_all(
    config: SyncConfig,
    manifest: Manifest,
    markdown_files: list[MarkdownFile],
) -> None:
    lark_cli.ensure_lark_cli()
    lark_cli.set_working_directory(config.repo_root)
    space_id = ensure_wiki_space(config, manifest)
    print(f"Wiki space: {manifest.wiki_space_name} ({space_id})")

    project_root = ensure_project_folder(space_id, config, manifest)
    print(f"Project folder: {manifest.project_folder_name} ({project_root})")

    for md_file in markdown_files:
        entry = ensure_file_node(space_id, project_root, manifest, md_file)
        sync_markdown_to_document(entry.document_id, md_file)
        entry.content_hash = md_file.content_hash
        manifest.set_file(md_file.relative_path, entry)
        print(f"Bootstrapped: {md_file.relative_path}")
