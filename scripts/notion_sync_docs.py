#!/usr/bin/env python3
"""Sync selected Notion pages into the local docs mirror.

This script is intentionally narrow. It mirrors a mapped set of Notion pages into
Markdown files under `docs/` so repository readers can consume the same canonical
content locally.

Requirements:
- `NOTION_TOKEN` must be set to a Notion integration token with access to the pages.
- `docs/notion-sync-map.json` must exist and define the pages to sync.

The renderer supports the subset of Notion blocks commonly used by Syncraft docs:
- paragraphs
- heading_1 / heading_2 / heading_3
- bulleted and numbered list items
- to-do items
- quotes
- callouts
- code blocks
- dividers
- toggles
- child pages
- bookmarks
"""

from __future__ import annotations

import argparse
import json
import os
import re
import sys
import textwrap
import urllib.error
import urllib.parse
import urllib.request
from dataclasses import dataclass
from pathlib import Path
from typing import Any


NOTION_API_BASE = "https://api.notion.com/v1"
DEFAULT_NOTION_VERSION = "2022-06-28"
DEFAULT_TIMEOUT_SECONDS = 20


class SyncError(RuntimeError):
    """Raised when the sync process cannot continue safely."""


@dataclass(frozen=True)
class SyncEntry:
    """Configuration for one Notion page mirrored to one local Markdown file."""

    title: str
    notion_url: str
    output_path: str

    @property
    def page_id(self) -> str:
        """Return the canonical Notion page id extracted from the URL."""

        match = re.search(r"([0-9a-fA-F]{32})", self.notion_url)
        if not match:
            raise SyncError(f"Could not extract page id from URL: {self.notion_url}")
        raw = match.group(1).lower()
        return (
            f"{raw[0:8]}-{raw[8:12]}-{raw[12:16]}-{raw[16:20]}-{raw[20:32]}"
        )


class NotionClient:
    """Very small Notion API client for page and block retrieval."""

    def __init__(self, token: str, notion_version: str, timeout_seconds: int) -> None:
        """Initialize the API client with auth and request settings."""

        if not token:
            raise SyncError("NOTION_TOKEN is required to sync docs.")
        self._token = token
        self._notion_version = notion_version
        self._timeout_seconds = timeout_seconds

    def get_page(self, page_id: str) -> dict[str, Any]:
        """Return Notion page metadata."""

        return self._request_json(f"/pages/{page_id}")

    def list_block_children(self, block_id: str) -> list[dict[str, Any]]:
        """Recursively retrieve the full direct-child tree for one block id."""

        results: list[dict[str, Any]] = []
        cursor: str | None = None
        while True:
            params = {}
            if cursor:
                params["start_cursor"] = cursor
            query = f"?{urllib.parse.urlencode(params)}" if params else ""
            payload = self._request_json(f"/blocks/{block_id}/children{query}")
            results.extend(payload.get("results", []))
            if not payload.get("has_more"):
                return results
            cursor = payload.get("next_cursor")

    def _request_json(self, path: str) -> dict[str, Any]:
        """Execute one JSON API request with explicit error handling."""

        url = f"{NOTION_API_BASE}{path}"
        request = urllib.request.Request(
            url,
            headers={
                "Authorization": f"Bearer {self._token}",
                "Notion-Version": self._notion_version,
                "Content-Type": "application/json",
            },
            method="GET",
        )
        try:
            with urllib.request.urlopen(
                request, timeout=self._timeout_seconds
            ) as response:
                return json.loads(response.read().decode("utf-8"))
        except urllib.error.HTTPError as exc:
            details = exc.read().decode("utf-8", errors="replace")
            raise SyncError(f"Notion API error {exc.code} for {url}: {details}") from exc
        except urllib.error.URLError as exc:
            raise SyncError(f"Network error while calling {url}: {exc}") from exc


def load_sync_map(config_path: Path) -> list[SyncEntry]:
    """Load and validate the sync configuration file."""

    if not config_path.exists():
        raise SyncError(f"Missing sync map: {config_path}")
    payload = json.loads(config_path.read_text(encoding="utf-8"))
    if not isinstance(payload, list):
        raise SyncError("Sync map must be a JSON array.")

    entries: list[SyncEntry] = []
    for item in payload:
        if not isinstance(item, dict):
            raise SyncError("Each sync map entry must be a JSON object.")
        entries.append(
            SyncEntry(
                title=str(item["title"]),
                notion_url=str(item["notion_url"]),
                output_path=str(item["output_path"]),
            )
        )
    return entries


def format_rich_text(rich_text: list[dict[str, Any]]) -> str:
    """Render Notion rich text into a compact Markdown string."""

    parts: list[str] = []
    for item in rich_text:
        text = item.get("plain_text", "")
        href = item.get("href")
        annotations = item.get("annotations", {})

        if href:
            text = f"[{text}]({href})"
        if annotations.get("code"):
            text = f"`{text}`"
        if annotations.get("bold"):
            text = f"**{text}**"
        if annotations.get("italic"):
            text = f"*{text}*"
        if annotations.get("strikethrough"):
            text = f"~~{text}~~"
        parts.append(text)
    return "".join(parts).strip()


def render_page_title(page_payload: dict[str, Any]) -> str:
    """Extract the title property from a page payload."""

    properties = page_payload.get("properties", {})
    for value in properties.values():
        if value.get("type") == "title":
            title = format_rich_text(value.get("title", []))
            if title:
                return title
    return "Untitled"


def render_blocks(
    client: NotionClient, blocks: list[dict[str, Any]], indent: int = 0
) -> list[str]:
    """Render a Notion block list into Markdown lines."""

    lines: list[str] = []
    for block in blocks:
        block_type = block.get("type", "")
        value = block.get(block_type, {})
        prefix = "  " * indent

        if block_type == "paragraph":
            text = format_rich_text(value.get("rich_text", []))
            if text:
                lines.append(f"{prefix}{text}")
                lines.append("")
        elif block_type == "heading_1":
            lines.append(f"{prefix}# {format_rich_text(value.get('rich_text', []))}")
            lines.append("")
        elif block_type == "heading_2":
            lines.append(f"{prefix}## {format_rich_text(value.get('rich_text', []))}")
            lines.append("")
        elif block_type == "heading_3":
            lines.append(f"{prefix}### {format_rich_text(value.get('rich_text', []))}")
            lines.append("")
        elif block_type == "bulleted_list_item":
            lines.append(f"{prefix}- {format_rich_text(value.get('rich_text', []))}")
        elif block_type == "numbered_list_item":
            lines.append(f"{prefix}1. {format_rich_text(value.get('rich_text', []))}")
        elif block_type == "to_do":
            checked = "x" if value.get("checked") else " "
            lines.append(
                f"{prefix}- [{checked}] {format_rich_text(value.get('rich_text', []))}"
            )
        elif block_type == "quote":
            lines.append(f"{prefix}> {format_rich_text(value.get('rich_text', []))}")
            lines.append("")
        elif block_type == "code":
            language = value.get("language", "text")
            lines.append(f"{prefix}```{language}")
            code_text = format_rich_text(value.get("rich_text", []))
            if code_text:
                for line in code_text.splitlines():
                    lines.append(f"{prefix}{line}")
            lines.append(f"{prefix}```")
            lines.append("")
        elif block_type == "divider":
            lines.append(f"{prefix}---")
            lines.append("")
        elif block_type == "callout":
            text = format_rich_text(value.get("rich_text", []))
            lines.append(f"{prefix}> {text}")
            lines.append("")
        elif block_type == "toggle":
            summary = format_rich_text(value.get("rich_text", []))
            lines.append(f"{prefix}<details>")
            lines.append(f"{prefix}<summary>{summary}</summary>")
            lines.append("")
            if block.get("has_children"):
                child_blocks = client.list_block_children(block["id"])
                lines.extend(render_blocks(client, child_blocks, indent + 1))
            lines.append(f"{prefix}</details>")
            lines.append("")
            continue
        elif block_type == "child_page":
            lines.append(f"{prefix}- {value.get('title', 'Child page')}")
        elif block_type == "bookmark":
            url = value.get("url", "")
            lines.append(f"{prefix}- {url}")
        else:
            lines.append(f"{prefix}<!-- Unsupported Notion block: {block_type} -->")
            lines.append("")

        if block.get("has_children") and block_type not in {"toggle"}:
            child_blocks = client.list_block_children(block["id"])
            child_lines = render_blocks(client, child_blocks, indent + 1)
            lines.extend(child_lines)

    return _squash_blank_lines(lines)


def _squash_blank_lines(lines: list[str]) -> list[str]:
    """Collapse repeated blank lines to keep Markdown readable."""

    squashed: list[str] = []
    previous_blank = False
    for line in lines:
        is_blank = line == ""
        if is_blank and previous_blank:
            continue
        squashed.append(line)
        previous_blank = is_blank
    return squashed


def render_document(
    client: NotionClient, entry: SyncEntry, page_payload: dict[str, Any]
) -> str:
    """Render one full Markdown document from Notion page data."""

    title = render_page_title(page_payload)
    blocks = client.list_block_children(entry.page_id)
    body_lines = render_blocks(client, blocks)
    header = textwrap.dedent(
        f"""\
        <!--
        Generated by scripts/notion_sync_docs.py
        Source: {entry.notion_url}
        -->

        # {title}

        """
    )
    body = "\n".join(body_lines).strip()
    return f"{header}{body}\n"


def sync_entry(
    client: NotionClient, repo_root: Path, entry: SyncEntry, dry_run: bool
) -> str:
    """Sync one entry and return a human-readable status line."""

    page_payload = client.get_page(entry.page_id)
    content = render_document(client, entry, page_payload)
    output_path = repo_root / entry.output_path
    output_path.parent.mkdir(parents=True, exist_ok=True)
    if not dry_run:
        output_path.write_text(content, encoding="utf-8")
    return f"{'DRY-RUN ' if dry_run else ''}synced {entry.title} -> {entry.output_path}"


def parse_args() -> argparse.Namespace:
    """Parse command-line arguments."""

    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "command",
        choices=["sync", "check"],
        help="sync mapped pages or validate the sync map",
    )
    parser.add_argument(
        "--config",
        default="docs/notion-sync-map.json",
        help="Path to the JSON sync map relative to repo root.",
    )
    parser.add_argument(
        "--only",
        action="append",
        default=[],
        help="Sync only entries whose output path matches one of these values.",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Render and validate without writing files.",
    )
    return parser.parse_args()


def main() -> int:
    """Run the CLI entrypoint with explicit failure reporting."""

    try:
        args = parse_args()
        repo_root = Path(__file__).resolve().parent.parent
        config_path = (repo_root / args.config).resolve()
        entries = load_sync_map(config_path)

        selected = entries
        if args.only:
            wanted = set(args.only)
            selected = [entry for entry in entries if entry.output_path in wanted]
            if not selected:
                raise SyncError("No sync-map entries matched --only filters.")

        if args.command == "check":
            for entry in selected:
                print(f"ok {entry.title} -> {entry.output_path} ({entry.page_id})")
            return 0

        client = NotionClient(
            token=os.environ.get("NOTION_TOKEN", ""),
            notion_version=os.environ.get("NOTION_VERSION", DEFAULT_NOTION_VERSION),
            timeout_seconds=DEFAULT_TIMEOUT_SECONDS,
        )
        for entry in selected:
            print(sync_entry(client, repo_root, entry, args.dry_run))
        return 0
    except SyncError as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
