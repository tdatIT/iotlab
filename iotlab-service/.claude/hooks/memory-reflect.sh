#!/usr/bin/env bash
set -uo pipefail
# UserPromptSubmit — when the user is correcting a mistake, remind the agent to capture the lesson.
#
# This runs on EVERY prompt, so it prints nothing unless the prompt actually carries a correction
# signal — an unconditional reminder would cost tokens on every single turn. It never exits 2:
# blocking a user's prompt over a memory reminder is never right.
#
# Skill: .agents/skills/project-memory/SKILL.md

PAYLOAD="$(cat)"
ROOT="${CLAUDE_PROJECT_DIR:-.}"
MEMORY_SH="$ROOT/.agents/skills/project-memory/scripts/memory.sh"
[[ -f "$MEMORY_SH" ]] || exit 0

read_prompt() {
    if command -v jq >/dev/null 2>&1; then
        printf '%s' "$PAYLOAD" | jq -r '.prompt // empty' 2>/dev/null
    else
        printf '%s' "$PAYLOAD" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("prompt",""))' 2>/dev/null
    fi
}

PROMPT="$(read_prompt | tr '[:upper:]' '[:lower:]')"
[[ -z "$PROMPT" ]] && exit 0

SIGNALS='sai rồi|sai r|không phải|ko phải|nhầm rồi|đừng |đã bảo|lần sau|rút kinh nghiệm|nhớ (là|cho)|tại sao lại|vì sao lại|thats wrong|that.s wrong|you were wrong|not what i|why did you|i told you|next time|remember (this|that)|stop doing'
printf '%s' "$PROMPT" | grep -qE "$SIGNALS" || exit 0

cat <<TXT
The user is correcting something. Before continuing, consider recording the lesson —
but only if it passes all four gates in .agents/skills/project-memory/SKILL.md:
durable, non-obvious, reusable, actionable. If any is doubtful, do NOT record it.

  bash $MEMORY_SH add --type lesson --title "..." --desc "..." --why "..." --apply "..."
TXT
exit 0
