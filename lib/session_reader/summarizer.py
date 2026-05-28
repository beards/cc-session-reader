"""Static extraction of tool call summaries and user answer detection."""

import json

USER_ANSWER_PREFIXES = (
    "User has answered your questions:",
    "Your questions have been answered:",
)


def summarize_tool_use(name: str, inp: dict) -> str:
    if name == "Bash":
        desc = inp.get("description", "")
        if desc:
            return f"[Bash] {desc}"
        return f"[Bash] {inp.get('command', '?')[:80]}"
    if name == "Read":
        path = inp.get("file_path", "?")
        parts = path.rsplit("/", 2)
        short = "/".join(parts[-2:]) if len(parts) >= 2 else path
        return f"[Read] {short}"
    if name == "Edit":
        return f"[Edit] {inp.get('file_path', '?').rsplit('/', 1)[-1]}"
    if name == "Write":
        return f"[Write] {inp.get('file_path', '?').rsplit('/', 1)[-1]}"
    if name == "Agent":
        desc = inp.get("description", "?")
        sub = inp.get("subagent_type", "")
        return f"[Agent({sub})] {desc}" if sub else f"[Agent] {desc}"
    if name == "Grep":
        pat = inp.get("pattern", "?")
        path = inp.get("path", "")
        return f'[Grep] "{pat}" in {path}' if path else f'[Grep] "{pat}"'
    if name == "Glob":
        return f"[Glob] {inp.get('pattern', '?')}"
    if name == "Skill":
        return f"[Skill] /{inp.get('skill', '?')} {inp.get('args', '')}".strip()[:80]
    if name == "AskUserQuestion":
        qs = inp.get("questions", [])
        if not qs or not isinstance(qs, list):
            return "[AskUserQuestion]"
        lines = [f"[AskUserQuestion] Q{i+1}: {q.get('question', '?')}"[:90] for i, q in enumerate(qs)]
        return "\n  ".join(lines)
    if name == "ToolSearch":
        return f"[ToolSearch] {inp.get('query', '?')}"
    return f"[{name}]"


def summarize_tool_result(entry: dict) -> str:
    tr = entry.get("toolUseResult", {})
    if not isinstance(tr, dict):
        return ""
    success = tr.get("success", True)
    status = "ok" if success else "FAILED"
    content = entry.get("message", {}).get("content", [])
    first_line = ""
    if isinstance(content, list):
        for block in content:
            if isinstance(block, dict) and block.get("type") == "tool_result":
                sub = block.get("content", "")
                if isinstance(sub, str) and sub.strip():
                    first_line = sub.strip().split("\n")[0][:80]
                    break
    return f" -> {status}: {first_line}" if first_line else f" -> {status}"


def is_user_answer(entry: dict) -> bool:
    content = entry.get("message", {}).get("content", [])
    if isinstance(content, list):
        for block in content:
            if isinstance(block, dict) and block.get("type") == "tool_result":
                sub = block.get("content", "")
                if isinstance(sub, str) and any(sub.startswith(p) for p in USER_ANSWER_PREFIXES):
                    return True
    return False


def extract_user_answers(entry: dict) -> str:
    content = entry.get("message", {}).get("content", [])
    if isinstance(content, list):
        for block in content:
            if isinstance(block, dict) and block.get("type") == "tool_result":
                sub = block.get("content", "")
                if isinstance(sub, str) and any(sub.startswith(p) for p in USER_ANSWER_PREFIXES):
                    return sub
    return ""
