#!/usr/bin/env bash
set -uo pipefail
# PostToolUse(Edit|Write) — check the comments of the .go file just written against go-style-core.
#
# Runs the skill's checker on that one file. A violation exits 2, which feeds the findings back to
# the agent so it fixes the comment before moving on. The file itself is already written; this
# hook never edits it.
#
# Rules: .agents/skills/go-style-core/SKILL.md ("Comments")

PAYLOAD="$(cat)"
ROOT="${CLAUDE_PROJECT_DIR:-.}"
CHECKER="$ROOT/.agents/skills/go-style-core/scripts/check-comments.sh"

read_path() {
    if command -v jq >/dev/null 2>&1; then
        printf '%s' "$PAYLOAD" | jq -r '.tool_input.file_path // empty' 2>/dev/null
    else
        printf '%s' "$PAYLOAD" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("file_path",""))' 2>/dev/null
    fi
}

FILE="$(read_path)"
[[ -z "$FILE" || "$FILE" != *.go || ! -f "$FILE" ]] && exit 0
[[ -f "$CHECKER" ]] || exit 0

OUT="$(bash "$CHECKER" "$FILE" 2>&1)"; RC=$?
[[ $RC -eq 0 ]] && exit 0

{
    echo "comment-rules: $FILE breaks the comment rules of go-style-core."
    echo "$OUT"
    echo "Fix these before continuing (C1 = line > 130 chars, C2 = > 2 comment lines in a function body,"
    echo "C3 = doc comment must start with the identifier name)."
} >&2
exit 2
