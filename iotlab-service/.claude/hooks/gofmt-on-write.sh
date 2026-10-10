#!/usr/bin/env bash
set -uo pipefail
# PostToolUse(Edit|Write) — gofmt any .go file the agent just wrote.
# Reads the hook payload on stdin; a non-Go path or a parse failure is a silent no-op.
# Never blocks: formatting is a convenience, not a gate.

PAYLOAD="$(cat)"

read_path() {
    if command -v jq >/dev/null 2>&1; then
        printf '%s' "$PAYLOAD" | jq -r '.tool_input.file_path // empty' 2>/dev/null
    else
        printf '%s' "$PAYLOAD" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' 2>/dev/null
    fi
}

FILE="$(read_path)"
[[ -z "$FILE" || "$FILE" != *.go || ! -f "$FILE" ]] && exit 0
command -v gofmt >/dev/null 2>&1 || exit 0

[[ -z "$(gofmt -l "$FILE" 2>/dev/null)" ]] && exit 0
gofmt -w "$FILE" 2>/dev/null && echo "gofmt: formatted $FILE"
exit 0
