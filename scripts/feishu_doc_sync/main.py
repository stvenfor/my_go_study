"""CLI for Feishu document sync (lark-cli backend)."""

from __future__ import annotations

import argparse
import os
import sys
from dataclasses import replace
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from bootstrap import (  # noqa: E402
    bootstrap_all,
    ensure_file_node,
    ensure_project_folder,
    ensure_wiki_space,
    sync_markdown_to_document,
)
import lark_cli
from lark_cli import LarkCliError, ensure_lark_cli  # noqa: E402
from manifest import Manifest  # noqa: E402
from scanner import load_config, scan_markdown_files  # noqa: E402


def _apply_repo_paths(config, repo_root: Path):
    return replace(
        config,
        repo_root=repo_root,
        manifest_path=repo_root / "docs" / "feishu-sync.manifest.json",
    )


def _repo_root() -> Path:
    return Path(__file__).resolve().parents[2]


def _resolve_config_path(explicit: str | None) -> Path:
    if explicit:
        return Path(explicit).resolve()
    return _repo_root() / "docs" / "feishu-sync.config.yaml"


def cmd_bootstrap(args: argparse.Namespace) -> int:
    config_path = _resolve_config_path(args.config)
    config = _apply_repo_paths(load_config(config_path), _repo_root())

    markdown_files = scan_markdown_files(config)
    if args.file:
        markdown_files = [item for item in markdown_files if item.relative_path == args.file]
        if not markdown_files:
            raise LarkCliError(f"Markdown file not in sync scope: {args.file}")

    if args.dry_run:
        print(f"[dry-run] Would bootstrap {len(markdown_files)} markdown files.")
        print(f"[dry-run] Wiki: {config.wiki_space_name} / {config.project_folder}")
        for item in markdown_files:
            print(f"  - {item.relative_path}")
        return 0

    ensure_lark_cli()
    lark_cli.set_working_directory(config.repo_root)
    manifest = Manifest.load(config.manifest_path)
    bootstrap_all(config, manifest, markdown_files)
    manifest.save(config.manifest_path)
    print(f"Manifest saved: {config.manifest_path}")
    return 0


def cmd_sync(args: argparse.Namespace) -> int:
    config_path = _resolve_config_path(args.config)
    config = _apply_repo_paths(load_config(config_path), _repo_root())

    markdown_files = scan_markdown_files(config)
    if args.file:
        markdown_files = [item for item in markdown_files if item.relative_path == args.file]
        if not markdown_files:
            raise LarkCliError(f"Markdown file not in sync scope: {args.file}")

    manifest = Manifest.load(config.manifest_path)

    if args.dry_run:
        changed = 0
        for md_file in markdown_files:
            entry = manifest.get_file(md_file.relative_path)
            if entry is None or entry.content_hash != md_file.content_hash:
                print(f"[dry-run] Would sync: {md_file.relative_path}")
                changed += 1
        print(f"[dry-run] {changed} file(s) would be updated.")
        return 0

    ensure_lark_cli()
    lark_cli.set_working_directory(config.repo_root)
    manifest = Manifest.load(config.manifest_path)

    if not manifest.wiki_space_id and not os.environ.get("FEISHU_WIKI_SPACE_ID"):
        print("Manifest has no wiki_space_id. Running bootstrap first...")
        bootstrap_all(config, manifest, markdown_files)
        manifest.save(config.manifest_path)
        return 0

    space_id = ensure_wiki_space(config, manifest)
    project_root = manifest.project_folder_node_token
    if not project_root:
        project_root = ensure_project_folder(space_id, config, manifest)

    changed = 0
    for md_file in markdown_files:
        entry = manifest.get_file(md_file.relative_path)
        needs_update = entry is None or entry.content_hash != md_file.content_hash
        if not needs_update:
            continue

        if entry is None or not entry.document_id:
            entry = ensure_file_node(space_id, project_root, manifest, md_file)

        sync_markdown_to_document(entry.document_id, md_file)
        entry.content_hash = md_file.content_hash
        manifest.set_file(md_file.relative_path, entry)
        changed += 1
        print(f"Synced: {md_file.relative_path}")

    manifest.save(config.manifest_path)
    print(f"Done. Updated {changed} file(s).")
    return 0


def build_parser() -> argparse.ArgumentParser:
    shared = argparse.ArgumentParser(add_help=False)
    shared.add_argument(
        "--config",
        help="Path to feishu-sync.config.yaml (default: docs/feishu-sync.config.yaml)",
    )
    shared.add_argument("--file", help="Sync a single markdown file (repo-relative path).")
    shared.add_argument("--dry-run", action="store_true", help="Print actions without calling Feishu.")

    parser = argparse.ArgumentParser(description="Sync Git Markdown docs to Feishu Wiki via lark-cli.")
    subparsers = parser.add_subparsers(dest="command", required=True)
    subparsers.add_parser(
        "bootstrap",
        parents=[shared],
        help="Create wiki space/project folder/nodes and import all markdown files.",
    )
    subparsers.add_parser(
        "sync",
        parents=[shared],
        help="Incremental sync changed markdown files.",
    )
    return parser


def main(argv: list[str] | None = None) -> int:
    parser = build_parser()
    args = parser.parse_args(argv)
    try:
        if args.command == "bootstrap":
            return cmd_bootstrap(args)
        if args.command == "sync":
            return cmd_sync(args)
        parser.error(f"Unknown command: {args.command}")
    except LarkCliError as exc:
        print(f"Error: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
