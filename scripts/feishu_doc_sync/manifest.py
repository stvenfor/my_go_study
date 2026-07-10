"""Manifest persistence for Feishu node mappings."""

from __future__ import annotations

import json
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any


@dataclass
class FileEntry:
    node_token: str
    document_id: str
    content_hash: str = ""


@dataclass
class Manifest:
    wiki_space_id: str = ""
    wiki_space_name: str = ""
    project_folder_name: str = ""
    project_folder_node_token: str = ""
    directories: dict[str, str] = field(default_factory=dict)
    files: dict[str, FileEntry] = field(default_factory=dict)

    @classmethod
    def load(cls, path: Path) -> Manifest:
        if not path.exists():
            return cls()
        with path.open(encoding="utf-8") as handle:
            raw: dict[str, Any] = json.load(handle)

        files: dict[str, FileEntry] = {}
        for key, value in raw.get("files", {}).items():
            files[key] = FileEntry(
                node_token=value.get("node_token", ""),
                document_id=value.get("document_id", ""),
                content_hash=value.get("content_hash", ""),
            )

        return cls(
            wiki_space_id=raw.get("wiki_space_id", ""),
            wiki_space_name=raw.get("wiki_space_name", ""),
            project_folder_name=raw.get("project_folder_name", ""),
            project_folder_node_token=raw.get("project_folder_node_token", ""),
            directories=dict(raw.get("directories", {})),
            files=files,
        )

    def save(self, path: Path) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        payload = {
            "wiki_space_id": self.wiki_space_id,
            "wiki_space_name": self.wiki_space_name,
            "project_folder_name": self.project_folder_name,
            "project_folder_node_token": self.project_folder_node_token,
            "directories": self.directories,
            "files": {
                key: {
                    "node_token": entry.node_token,
                    "document_id": entry.document_id,
                    "content_hash": entry.content_hash,
                }
                for key, entry in sorted(self.files.items())
            },
        }
        with path.open("w", encoding="utf-8") as handle:
            json.dump(payload, handle, ensure_ascii=False, indent=2)
            handle.write("\n")

    def get_directory_node(self, directory_key: str) -> str | None:
        return self.directories.get(directory_key)

    def set_directory_node(self, directory_key: str, node_token: str) -> None:
        self.directories[directory_key] = node_token

    def get_file(self, relative_path: str) -> FileEntry | None:
        return self.files.get(relative_path)

    def set_file(self, relative_path: str, entry: FileEntry) -> None:
        self.files[relative_path] = entry
