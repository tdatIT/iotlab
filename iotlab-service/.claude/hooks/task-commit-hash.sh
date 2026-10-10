#!/usr/bin/env bash
set -uo pipefail
# PostToolUse(Bash) — after a successful `git commit`, hand over the fresh hash and the exact
# command that closes the open task. The task-management gate needs a real commit hash, and the
# hash only exists at this moment: commit first, complete after.
#
# Advisory only (always exit 0). Silent when no task is in progress.
#
# Skill: .agents/skills/task-management/SKILL.md

PAYLOAD="$(cat)"
ROOT="${CLAUDE_PROJECT_DIR:-.}"
TASK_SH="$ROOT/.agents/skills/task-management/scripts/task.sh"
[[ -f "$TASK_SH" && -d "$ROOT/.agents/tasks" ]] || exit 0

read_cmd() {
    if command -v jq >/dev/null 2>&1; then
        printf '%s' "$PAYLOAD" | jq -r '.tool_input.command // empty' 2>/dev/null
    else
        printf '%s' "$PAYLOAD" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("tool_input",{}).get("command",""))' 2>/dev/null
    fi
}

CMD="$(read_cmd)"
[[ -z "$CMD" ]] && exit 0
printf '%s' "$CMD" | grep -qE '(^|[;&|]|&&)[[:space:]]*git[[:space:]]+commit\b' || exit 0
# `git commit --amend` rewrites a commit that is already recorded on the task.
printf '%s' "$CMD" | grep -q -- '--amend' && exit 0

cd "$ROOT" 2>/dev/null || exit 0
HASH="$(git rev-parse --short HEAD 2>/dev/null)"
[[ -z "$HASH" ]] && exit 0

fm() { sed -n '/^---$/,/^---$/p' "$1" | grep -E "^$2:" | head -1 | sed -E "s/^$2:[[:space:]]*//"; }

for f in .agents/tasks/TASK-*.md; do
    [[ -e "$f" ]] || continue
    [[ "$(fm "$f" status)" == "in_progress" ]] || continue
    ID="$(fm "$f" id)"
    BRANCH="$(fm "$f" branch)"
    LEFT="$(grep -c '^- \[ \]' "$f" || true)"
    echo "task: $ID is in progress; commit $HASH was just created."
    if [[ "${LEFT:-0}" -gt 0 ]]; then
        echo "  ${LEFT} checklist item(s) still unticked in $f — tick them before completing."
    fi
    echo "  bash $TASK_SH complete --id $ID --summary \"<what you did>\" --commit $HASH${BRANCH:+ --branch $BRANCH}"
done
exit 0
