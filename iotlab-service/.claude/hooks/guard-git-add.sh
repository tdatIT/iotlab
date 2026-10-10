#!/usr/bin/env bash
set -uo pipefail
# PreToolUse(Bash) — block blanket staging (`git add -A`, `git add .`, `git add --all`).
# Blanket staging sweeps up junk: local config, scratch files, half-finished work, and anything
# .gitignore does not happen to cover. Stage the paths you actually changed.
# exit 2 = deny the tool call; stderr goes back to the agent.

PAYLOAD="$(cat)"

read_cmd() {
    if command -v jq >/dev/null 2>&1; then
        printf '%s' "$PAYLOAD" | jq -r '.tool_input.command // empty' 2>/dev/null
    else
        printf '%s' "$PAYLOAD" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("command",""))' 2>/dev/null
    fi
}

CMD="$(read_cmd)"
[[ -z "$CMD" ]] && exit 0

# Match `git add` with -A / --all / bare `.` as a whole argument, anywhere in a compound command.
if printf '%s' "$CMD" | grep -qE '(^|[;&|]|&&)[[:space:]]*git[[:space:]]+add[[:space:]]+([^;&|]*[[:space:]])?(-A|--all|\.)([[:space:]]|$)'; then
    echo "guard-git-add: blanket staging is blocked ('git add -A' / 'git add .' / 'git add --all')." >&2
    echo "Stage explicit paths instead, e.g. 'git add internal/app/product/command/create_product.go'." >&2
    echo "Run 'git status --short' first to see exactly what changed." >&2
    exit 2
fi
exit 0
