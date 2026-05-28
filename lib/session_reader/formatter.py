"""Output formatters for read and context modes."""

import json
import os
import sys
from pathlib import Path

from .parser import (
    collect_agent_tool_ids,
    extract_text,
    extract_tool_result_text,
    format_timestamp,
    get_tool_uses,
    is_noise,
    load_session_meta,
)
from .summarizer import (
    extract_user_answers,
    is_user_answer,
    summarize_tool_result,
    summarize_tool_use,
)


def format_read(
    transcript: Path,
    session_id: str,
    *,
    max_lines: int = 0,
    verbose_agents: bool = False,
    output=None,
) -> None:
    out = output or sys.stdout
    agent_ids = collect_agent_tool_ids(transcript) if verbose_agents else set()

    lines_output = 0
    pending_tools: list[str] = []

    def flush():
        nonlocal lines_output
        for s in pending_tools:
            out.write(f"  {s}\n")
            lines_output += 1
        if pending_tools:
            out.write("\n")
            lines_output += 1
        pending_tools.clear()

    with open(transcript) as f:
        for raw_line in f:
            if max_lines and lines_output >= max_lines:
                out.write(f"\n--- truncated at {max_lines} output lines ---\n")
                break

            entry = _safe_parse(raw_line)
            if entry is None or is_noise(entry):
                continue

            message = entry.get("message", {})
            if not isinstance(message, dict):
                continue

            if "toolUseResult" in entry:
                _handle_tool_result_read(
                    entry, agent_ids, pending_tools, flush, out, lines_output_ref=[lines_output],
                )
                lines_output = lines_output_ref[0] if False else lines_output
                continue

            role = message.get("role", "")
            content = message.get("content")
            ts = format_timestamp(entry.get("timestamp", ""))

            if role == "user":
                flush()
                text = extract_text(content)
                if not text.strip():
                    continue
                out.write(f"[{ts}] user:\n{text}\n\n")
                lines_output += text.count("\n") + 3

            elif role == "assistant":
                text = extract_text(content)
                tool_blocks = get_tool_uses(content)

                if not text.strip() and not tool_blocks:
                    continue

                if text.strip():
                    flush()
                    out.write(f"[{ts}] assistant:\n{text}\n")
                    lines_output += text.count("\n") + 2

                for tb in tool_blocks:
                    pending_tools.append(
                        summarize_tool_use(tb.get("name", "?"), tb.get("input", {}))
                    )

                if text.strip() and not tool_blocks:
                    out.write("\n")
                    lines_output += 1

    flush()


def _handle_tool_result_read(entry, agent_ids, pending_tools, flush_fn, out, lines_output_ref):
    if is_user_answer(entry):
        flush_fn()
        ts = format_timestamp(entry.get("timestamp", ""))
        answer = extract_user_answers(entry)
        out.write(f"[{ts}] user (answer):\n{answer}\n\n")
    elif agent_ids:
        full_text, tool_use_id = extract_tool_result_text(entry)
        if tool_use_id in agent_ids and full_text.strip():
            flush_fn()
            ts = format_timestamp(entry.get("timestamp", ""))
            out.write(f"[{ts}] agent result:\n{full_text}\n\n")
        else:
            result_str = summarize_tool_result(entry)
            if pending_tools:
                pending_tools[-1] += result_str
    else:
        result_str = summarize_tool_result(entry)
        if pending_tools:
            pending_tools[-1] += result_str


def format_context(
    transcript: Path,
    session_id: str,
    *,
    verbose_agents: bool = False,
    output=None,
) -> None:
    out = output or sys.stdout
    agent_ids = collect_agent_tool_ids(transcript) if verbose_agents else set()

    meta = load_session_meta(session_id)
    if meta:
        project = os.path.basename(meta.get("project_path", "?"))
        duration = meta.get("duration_minutes", "?")
        out.write(f"# Session {session_id[:8]} | {project} | {duration}m\n\n")

    pending_tools: list[str] = []

    def flush():
        for s in pending_tools:
            out.write(f"  {s}\n")
        if pending_tools:
            out.write("\n")
        pending_tools.clear()

    with open(transcript) as f:
        for raw_line in f:
            entry = _safe_parse(raw_line)
            if entry is None or is_noise(entry):
                continue

            message = entry.get("message", {})
            if not isinstance(message, dict):
                continue

            if "toolUseResult" in entry:
                if is_user_answer(entry):
                    flush()
                    out.write(f"U (answer): {extract_user_answers(entry)}\n\n")
                elif agent_ids:
                    full_text, tool_use_id = extract_tool_result_text(entry)
                    if tool_use_id in agent_ids and full_text.strip():
                        flush()
                        out.write(f"Agent result:\n{full_text}\n\n")
                    else:
                        result_str = summarize_tool_result(entry)
                        if pending_tools:
                            pending_tools[-1] += result_str
                else:
                    result_str = summarize_tool_result(entry)
                    if pending_tools:
                        pending_tools[-1] += result_str
                continue

            role = message.get("role", "")
            content = message.get("content")

            if role == "user":
                flush()
                text = extract_text(content)
                if text.strip():
                    out.write(f"U: {text}\n\n")

            elif role == "assistant":
                text = extract_text(content)
                tool_blocks = get_tool_uses(content)

                if text.strip():
                    flush()
                    out.write(f"A: {text}\n\n")

                for tb in tool_blocks:
                    pending_tools.append(
                        summarize_tool_use(tb.get("name", "?"), tb.get("input", {}))
                    )

    flush()


def _safe_parse(raw_line: str) -> dict | None:
    try:
        return json.loads(raw_line)
    except json.JSONDecodeError:
        return None
