"""Token counting via Anthropic API or heuristic estimation."""


def estimate_tokens(text: str) -> int:
    cjk = sum(
        1 for ch in text
        if 0x4E00 <= ord(ch) <= 0x9FFF or 0x3400 <= ord(ch) <= 0x4DBF
    )
    ascii_count = len(text) - cjk
    return int(cjk * 1.5 + ascii_count * 0.25)


def count_tokens_api(text: str) -> int | None:
    try:
        import anthropic
        client = anthropic.Anthropic()
        result = client.messages.count_tokens(
            model="claude-sonnet-4-6",
            messages=[{"role": "user", "content": text}],
        )
        return result.input_tokens
    except Exception:
        return None
