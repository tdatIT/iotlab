#!/usr/bin/env bash
set -uo pipefail
# SessionStart — recall: inject the project memory index and any in-progress task.
# stdout lands in the agent's context, so this runs once per session and stays short.
# Silent when there is nothing to recall: an empty header is pure token cost.
#
# Skills: .agents/skills/project-memory/SKILL.md · .agents/skills/task-management/SKILL.md

ROOT="${CLAUDE_PROJECT_DIR:-.}"
MEMORY_SH="$ROOT/.agents/skills/project-memory/scripts/memory.sh"
TASK_SH="$ROOT/.agents/skills/task-management/scripts/task.sh"
MAX_MEMORY_LINES=25

cd "$ROOT" 2>/dev/null || exit 0
OUT=""

# --- memory index (index rows start with "- [") ---
if [[ -f "$MEMORY_SH" && -d .agents/memory ]]; then
    ROWS="$(bash "$MEMORY_SH" list 2>/dev/null | grep '^- \[' || true)"
    COUNT="$(printf '%s' "$ROWS" | grep -c . || true)"
    if [[ "${COUNT:-0}" -gt 0 ]]; then
        OUT+="## Project memory (${COUNT})"$'\n'
        OUT+="$(printf '%s\n' "$ROWS" | head -n "$MAX_MEMORY_LINES")"$'\n'
        if [[ "$COUNT" -gt "$MAX_MEMORY_LINES" ]]; then
            OUT+="… $((COUNT - MAX_MEMORY_LINES)) more — bash $MEMORY_SH search <keyword>"$'\n'
        fi
        OUT+=$'\n'
    fi
fi

# --- tasks still open (read the files: status lives in their frontmatter) ---
if [[ -f "$TASK_SH" && -d .agents/tasks ]]; then
    OPEN=""
    for f in .agents/tasks/TASK-*.md; do
        [[ -e "$f" ]] || continue
        STATUS="$(sed -n '/^---$/,/^---$/p' "$f" | grep -E '^status:' | head -1 | sed -E 's/^status:[[:space:]]*//')"
        case "$STATUS" in
            in_progress|todo|blocked) ;;
            *) continue ;;
        esac
        ID="$(sed -n '/^---$/,/^---$/p' "$f" | grep -E '^id:' | head -1 | sed -E 's/^id:[[:space:]]*//')"
        TITLE="$(sed -n '/^---$/,/^---$/p' "$f" | grep -E '^title:' | head -1 | sed -E 's/^title:[[:space:]]*//')"
        BRANCH="$(sed -n '/^---$/,/^---$/p' "$f" | grep -E '^branch:' | head -1 | sed -E 's/^branch:[[:space:]]*//')"
        OPEN+="- ${ID} [${STATUS}] ${TITLE}${BRANCH:+ (branch: ${BRANCH})}"$'\n'
    done
    if [[ -n "$OPEN" ]]; then
        OUT+="## Open tasks"$'\n'"$OPEN"
        OUT+="Read the task before working on it: bash $TASK_SH show <id>"$'\n'
    fi
fi

[[ -z "$OUT" ]] && exit 0
printf '%s' "$OUT"
exit 0
