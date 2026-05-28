#!/usr/bin/env python3
"""CLI entry point for Claude session reader. Called from SKILL.md."""

import argparse
import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent.parent / "lib"))

from session_reader import (
    SESSION_META_DIR,
    count_tokens_api,
    estimate_tokens,
    extract_all_text,
    extract_user_answers,
    find_transcript,
    format_context,
    format_read,
    format_timestamp,
    get_tool_uses,
    is_noise,
    is_user_answer,
    parse_transcript,
    resolve_session_id,
    summarize_tool_result,
    summarize_tool_use,
)


def cmd_list(args):
    meta_files = sorted(
        SESSION_META_DIR.glob("*.json"),
        key=lambda f: f.stat().st_mtime,
        reverse=True,
    )
    project_filter = args.project.lower() if args.project else None
    printed = 0
    for mf in meta_files:
        if printed >= args.limit:
            break
        try:
            meta = json.loads(mf.read_text())
        except (json.JSONDecodeError, OSError):
            continue
        project = meta.get("project_path", "")
        project_name = os.path.basename(project) if project else "?"
        if project_filter and project_filter not in project_name.lower():
            continue
        sid = meta.get("session_id", mf.stem)
        start = meta.get("start_time", "")
        duration = meta.get("duration_minutes", 0)
        user_msgs = meta.get("user_message_count", 0)
        asst_msgs = meta.get("assistant_message_count", 0)
        first_prompt = meta.get("first_prompt", "")
        if len(first_prompt) > 80:
            first_prompt = first_prompt[:77] + "..."
        date_str = format_timestamp(start) if start else "??-??"
        print(f"{sid}  {date_str}  {project_name:<20}  {duration:>3}m  u:{user_msgs} a:{asst_msgs}  {first_prompt}")
        printed += 1
    if printed == 0:
        print("No sessions found.", file=sys.stderr)


def cmd_read(args):
    transcript = find_transcript(args.session_id)
    if not transcript:
        print(f"Transcript not found: {args.session_id}", file=sys.stderr)
        sys.exit(1)
    format_read(
        transcript, args.session_id,
        max_lines=args.max_lines,
        verbose_agents=args.verbose_agents,
    )


def cmd_context(args):
    transcript = find_transcript(args.session_id)
    if not transcript:
        print(f"Transcript not found: {args.session_id}", file=sys.stderr)
        sys.exit(1)
    format_context(
        transcript, args.session_id,
        verbose_agents=args.verbose_agents,
    )


def cmd_stats(args):
    transcript = find_transcript(args.session_id)
    if not transcript:
        print(f"Transcript not found: {args.session_id}", file=sys.stderr)
        sys.exit(1)

    entries = parse_transcript(transcript)
    raw_parts: list[str] = []
    filtered_parts: list[str] = []
    cat = {
        "user_text": 0, "user_answers": 0, "assistant_text": 0,
        "tool_summaries": 0, "tool_input_raw": 0, "tool_result_raw": 0,
        "system_noise": 0,
    }

    for entry in entries:
        message = entry.get("message", {})
        if not isinstance(message, dict):
            continue
        if is_noise(entry):
            text = extract_all_text(entry)
            cat["system_noise"] += len(text)
            raw_parts.append(text)
            continue

        content = message.get("content")

        if "toolUseResult" in entry:
            full = extract_all_text(entry)
            if is_user_answer(entry):
                answer = extract_user_answers(entry)
                cat["user_answers"] += len(answer)
                raw_parts.append(full)
                filtered_parts.append(answer)
            else:
                cat["tool_result_raw"] += len(full)
                raw_parts.append(full)
                summary = summarize_tool_result(entry)
                cat["tool_summaries"] += len(summary)
                filtered_parts.append(summary)
            continue

        role = message.get("role", "")
        if role == "user":
            from session_reader.parser import extract_text
            text = extract_text(content)
            if text.strip():
                cat["user_text"] += len(text)
                raw_parts.append(text)
                filtered_parts.append(text)
        elif role == "assistant":
            from session_reader.parser import extract_text
            text = extract_text(content)
            if text.strip():
                cat["assistant_text"] += len(text)
                raw_parts.append(text)
                filtered_parts.append(text)
            for tb in get_tool_uses(content):
                raw_json = json.dumps(tb.get("input", {}), ensure_ascii=False)
                cat["tool_input_raw"] += len(raw_json)
                raw_parts.append(raw_json)
                summary = summarize_tool_use(tb.get("name", "?"), tb.get("input", {}))
                cat["tool_summaries"] += len(summary)
                filtered_parts.append(summary)

    raw_text = "\n".join(raw_parts)
    filtered_text = "\n".join(filtered_parts)
    raw_c = len(raw_text)
    filt_c = len(filtered_text)

    print(f"Session: {args.session_id[:8]}")
    print(f"Transcript: {transcript.stat().st_size / 1024:.1f}KB\n")
    print("=== Characters ===")
    print(f"  Raw:      {raw_c:>10,}")
    print(f"  Filtered: {filt_c:>10,}")
    if raw_c > 0:
        saved = raw_c - filt_c
        print(f"  Saved:    {saved:>10,} ({saved * 100 / raw_c:.1f}%)")
    print(f"\n=== Breakdown ===")
    for label, key in [
        ("KEPT  user text:        ", "user_text"),
        ("KEPT  user answers:     ", "user_answers"),
        ("KEPT  assistant text:   ", "assistant_text"),
        ("KEPT  tool summaries:   ", "tool_summaries"),
        ("CUT   tool input (raw): ", "tool_input_raw"),
        ("CUT   tool result (raw):", "tool_result_raw"),
        ("CUT   system/noise:     ", "system_noise"),
    ]:
        print(f"  {label} {cat[key]:>10,}")

    if not args.no_tokens:
        print()
        raw_api = count_tokens_api(raw_text)
        filt_api = count_tokens_api(filtered_text)
        if raw_api is not None and filt_api is not None:
            saved = raw_api - filt_api
            print("=== Tokens (Anthropic API) ===")
            print(f"  Raw:      {raw_api:>10,}")
            print(f"  Filtered: {filt_api:>10,}")
            if raw_api > 0:
                print(f"  Saved:    {saved:>10,} ({saved * 100 / raw_api:.1f}%)")
        else:
            raw_est = estimate_tokens(raw_text)
            filt_est = estimate_tokens(filtered_text)
            saved_est = raw_est - filt_est
            print("=== Tokens (estimated) ===")
            print(f"  Raw:      {raw_est:>10,} ~")
            print(f"  Filtered: {filt_est:>10,} ~")
            if raw_est > 0:
                print(f"  Saved:    {saved_est:>10,} ~ ({saved_est * 100 / raw_est:.1f}%)")


def cmd_audit(args):
    transcript = find_transcript(args.session_id)
    if not transcript:
        print(f"Transcript not found: {args.session_id}", file=sys.stderr)
        sys.exit(1)

    entries = parse_transcript(transcript)
    limit = args.samples
    categories: dict[str, list[str]] = {
        "tool_result_cut": [], "system_noise": [], "thinking": [],
    }
    for entry in entries:
        message = entry.get("message", {})
        if not isinstance(message, dict):
            continue
        if is_noise(entry):
            text = extract_all_text(entry)
            if text.strip():
                categories["system_noise"].append(f"[{entry.get('type','')}] {text[:200]}")
            continue
        if "toolUseResult" in entry and not is_user_answer(entry):
            text = extract_all_text(entry)
            tr = entry.get("toolUseResult", {})
            name = tr.get("commandName", "?") if isinstance(tr, dict) else "?"
            if text.strip() and len(text) > 100:
                categories["tool_result_cut"].append(f"[{name}] {text[:300]}")
            continue
        if message.get("role") == "assistant" and isinstance(message.get("content"), list):
            for block in message["content"]:
                if isinstance(block, dict) and block.get("type") == "thinking":
                    t = block.get("thinking", "")
                    if t.strip():
                        categories["thinking"].append(t[:300])

    for cat_name, items in categories.items():
        if not items:
            continue
        shown = min(limit, len(items))
        print(f"=== {cat_name} ({len(items)} items, showing {shown}) ===")
        for item in items[:shown]:
            print(f"  {item}\n")
        if len(items) > shown:
            print(f"  ... and {len(items) - shown} more\n")


def main():
    parser = argparse.ArgumentParser(description="Claude session reader")
    sub = parser.add_subparsers(dest="command", required=True)

    p = sub.add_parser("list")
    p.add_argument("-n", "--limit", type=int, default=20)
    p.add_argument("-p", "--project", type=str)
    p.set_defaults(func=cmd_list)

    p = sub.add_parser("read")
    p.add_argument("session_id")
    p.add_argument("--max-lines", type=int, default=0)
    p.add_argument("--verbose-agents", action="store_true")
    p.set_defaults(func=cmd_read)

    p = sub.add_parser("context")
    p.add_argument("session_id")
    p.add_argument("--verbose-agents", action="store_true")
    p.set_defaults(func=cmd_context)

    p = sub.add_parser("stats")
    p.add_argument("session_id")
    p.add_argument("--no-tokens", action="store_true")
    p.set_defaults(func=cmd_stats)

    p = sub.add_parser("audit")
    p.add_argument("session_id")
    p.add_argument("-n", "--samples", type=int, default=5)
    p.set_defaults(func=cmd_audit)

    args = parser.parse_args()
    if hasattr(args, "session_id"):
        try:
            args.session_id = resolve_session_id(args.session_id)
        except ValueError as e:
            print(str(e), file=sys.stderr)
            sys.exit(1)
    args.func(args)


if __name__ == "__main__":
    main()
