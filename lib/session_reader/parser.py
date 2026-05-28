"""Transcript parsing, session discovery, and low-level content extraction."""

import json
from datetime import datetime
from pathlib import Path

CLAUDE_DIR = Path.home() / ".claude"
PROJECTS_DIR = CLAUDE_DIR / "projects"
SESSION_META_DIR = CLAUDE_DIR / "usage-data" / "session-meta"

NOISE_TYPES = frozenset({
    "file-history-snapshot", "attachment", "bridge-session",
    "last-prompt", "permission-mode", "ai-title", "queue-operation",
})


def find_transcript(session_id: str) -> Path | None:
    for jsonl in PROJECTS_DIR.rglob(f"{session_id}.jsonl"):
        return jsonl
    return None


def load_session_meta(session_id: str) -> dict | None:
    meta_file = SESSION_META_DIR / f"{session_id}.json"
    if meta_file.exists():
        return json.loads(meta_file.read_text())
    return None


def resolve_session_id(prefix: str) -> str:
    if len(prefix) == 36:
        return prefix
    matches = [j.stem for j in PROJECTS_DIR.rglob("*.jsonl") if j.stem.startswith(prefix)]
    if len(matches) == 1:
        return matches[0]
    if len(matches) > 1:
        raise ValueError(f"Ambiguous prefix '{prefix}', matches: {', '.join(m[:12] for m in matches[:5])}")
    return prefix


def parse_transcript(path: Path) -> list[dict]:
    entries = []
    with open(path) as f:
        for line in f:
            try:
                entries.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    return entries


def format_timestamp(ts_str: str) -> str:
    try:
        dt = datetime.fromisoformat(ts_str.replace("Z", "+00:00"))
        return dt.strftime("%m-%d %H:%M")
    except (ValueError, AttributeError):
        return "??-?? ??:??"


def extract_text(content) -> str:
    if isinstance(content, str):
        return content
    if isinstance(content, list):
        return "\n".join(
            b.get("text", "") for b in content
            if isinstance(b, dict) and b.get("type") == "text"
        )
    return ""


def get_tool_uses(content) -> list[dict]:
    if not isinstance(content, list):
        return []
    return [b for b in content if isinstance(b, dict) and b.get("type") == "tool_use"]


def collect_agent_tool_ids(path: Path) -> set[str]:
    ids = set()
    with open(path) as f:
        for line in f:
            try:
                e = json.loads(line)
            except json.JSONDecodeError:
                continue
            if e.get("type") != "assistant":
                continue
            content = e.get("message", {}).get("content", [])
            if not isinstance(content, list):
                continue
            for bl in content:
                if isinstance(bl, dict) and bl.get("type") == "tool_use" and bl.get("name") == "Agent":
                    ids.add(bl.get("id", ""))
    return ids


def extract_tool_result_text(entry: dict) -> tuple[str, str]:
    content = entry.get("message", {}).get("content", [])
    if not isinstance(content, list):
        return "", ""
    for bl in content:
        if not isinstance(bl, dict) or bl.get("type") != "tool_result":
            continue
        tool_use_id = bl.get("tool_use_id", "")
        sub = bl.get("content", "")
        if isinstance(sub, str):
            return sub, tool_use_id
        if isinstance(sub, list):
            text = "\n".join(
                b.get("text", "") for b in sub
                if isinstance(b, dict) and b.get("type") == "text"
            )
            return text, tool_use_id
    return "", ""


def extract_all_text(entry: dict) -> str:
    parts = []
    message = entry.get("message", {})
    if not isinstance(message, dict):
        return ""
    content = message.get("content")
    if isinstance(content, str):
        parts.append(content)
    elif isinstance(content, list):
        for block in content:
            if not isinstance(block, dict):
                continue
            if block.get("type") == "text":
                parts.append(block.get("text", ""))
            elif block.get("type") == "tool_use":
                parts.append(json.dumps(block.get("input", {}), ensure_ascii=False))
            elif block.get("type") == "tool_result":
                sub = block.get("content", "")
                if isinstance(sub, str):
                    parts.append(sub)
                elif isinstance(sub, list):
                    for sc in sub:
                        if isinstance(sc, dict) and sc.get("type") == "text":
                            parts.append(sc.get("text", ""))
    tr = entry.get("toolUseResult", {})
    if isinstance(tr, dict):
        for key in ("stdout", "stderr", "output"):
            if key in tr and tr[key]:
                parts.append(str(tr[key]))
    return "\n".join(parts)


def is_noise(entry: dict) -> bool:
    return entry.get("type", "") in NOISE_TYPES or entry.get("type", "") == "system"
