---
name: sessions
description: |
  讀取過去的 Claude Code session，靜態提取 tool call 摘要，過濾噪音但保留操作脈絡。
  觸發：/sessions、「看之前的對話」「那次討論了什麼」「回顧 session」。
allowed-tools:
  - Bash
  - Read
---

# Session Reader

讀取歷史 Claude Code session 的工具。每個 tool call 保留一行靜態摘要（tool name + key param + result 狀態），
過濾 tool result 的完整 stdout/stderr 和檔案內容。不截斷對話文字，不用 LLM。

典型 token reduction: **80-88%**。

## Script

```
SCRIPT=${SKILL_DIR}/scripts/sessions.py
```

## 子命令

### list — 列出 session

```bash
python3 ${SKILL_DIR}/scripts/sessions.py list -n 20
python3 ${SKILL_DIR}/scripts/sessions.py list -p zeroclaw -n 10
```

### read — 讀取對話 + inline tool 摘要

```bash
python3 ${SKILL_DIR}/scripts/sessions.py read <session-id>
python3 ${SKILL_DIR}/scripts/sessions.py read <session-id> --max-lines 200
python3 ${SKILL_DIR}/scripts/sessions.py read <session-id> --verbose-agents
```

`--verbose-agents`：完整保留 Agent subagent 回傳的分析結果（預設只保留一行摘要）。
用於優化 skill/agent prompt 時開啟。

### context — 精簡注入格式

```bash
python3 ${SKILL_DIR}/scripts/sessions.py context <session-id>
python3 ${SKILL_DIR}/scripts/sessions.py context <session-id> --verbose-agents
```

### stats — token 節省統計

```bash
python3 ${SKILL_DIR}/scripts/sessions.py stats <session-id>
python3 ${SKILL_DIR}/scripts/sessions.py stats <session-id> --no-tokens
```

有 `ANTHROPIC_API_KEY` 時用 API 精確計算 token，否則用 heuristic 估算。

### audit — 檢視被移除的內容

```bash
python3 ${SKILL_DIR}/scripts/sessions.py audit <session-id> -n 10
```

## 使用流程

1. `list` 找到目標 session
2. `read` 或 `context` 讀取內容（session ID 支援 prefix match，前 8 碼通常就夠）
3. 需要看 Agent 分析細節時加 `--verbose-agents`

## 保留什麼 / 過濾什麼

| 保留 | 過濾 |
|------|------|
| User 對話文字 | Tool result stdout/stderr |
| Assistant 對話文字 | 檔案全文 (Read/Edit content) |
| User answers (AskUserQuestion) | 背景任務確認 |
| Tool call 一行摘要 (name + key param + status) | Tool input 完整 JSON |
| Agent results (--verbose-agents) | System/noise messages |
