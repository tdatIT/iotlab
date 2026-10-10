#!/usr/bin/env bash
set -uo pipefail
# PreToolUse(Bash) — enforce the writing-commit-message skill on every `git commit`.
#
# Extracts the message from the command (-m, repeated -m, --message=, heredoc, -F file) and runs
# the skill's validator. A rule violation DENIES the commit (exit 2) and the reason goes back to
# the agent. When no message can be extracted (editor mode, --amend --no-edit) the commit is
# allowed with a pointer to the skill — this hook never guesses a message.
#
# Skill: .agents/skills/writing-commit-message/SKILL.md

PAYLOAD="$(cat)"
ROOT="${CLAUDE_PROJECT_DIR:-.}"
SKILL="$ROOT/.agents/skills/writing-commit-message"
VALIDATOR="$SKILL/scripts/check-commit-msg.sh"

# No python3 -> cannot parse the command safely; never block a commit over tooling.
command -v python3 >/dev/null 2>&1 || exit 0

# Prints one of: "SKIP" | "NOMSG" | "MSG<TAB><message>" | "FILE<TAB><path>"
# The heredoc is python's program, so the payload is passed as argv[1], not on stdin.
PARSED="$(python3 - "$PAYLOAD" <<'PY'
import json, re, shlex, sys

try:
    cmd = json.loads(sys.argv[1]).get("tool_input", {}).get("command", "")
except Exception:
    print("SKIP"); sys.exit(0)

if not re.search(r'(^|[;&|(]|&&)\s*git\s+(-\S+\s+|--\S+\s+)*commit\b', cmd):
    print("SKIP"); sys.exit(0)

# Reword/amend without a new message: nothing to validate.
if "--amend" in cmd and ("--no-edit" in cmd or "--reuse-message" in cmd):
    print("SKIP"); sys.exit(0)

# Heredoc form: git commit -m "$(cat <<'EOF' ... EOF)"
m = re.search(r"<<-?\s*['\"]?(\w+)['\"]?\r?\n(.*?)\r?\n\s*\1", cmd, re.S)
if m:
    print("MSG\t" + m.group(2)); sys.exit(0)

try:
    parts = shlex.split(cmd, comments=False)
except ValueError:
    print("NOMSG"); sys.exit(0)

msgs, path, i = [], None, 0
while i < len(parts):
    p = parts[i]
    if p in ("-m", "--message"):
        if i + 1 < len(parts):
            msgs.append(parts[i + 1]); i += 2; continue
    elif p.startswith("--message="):
        msgs.append(p.split("=", 1)[1]); i += 1; continue
    elif p in ("-F", "--file"):
        if i + 1 < len(parts):
            path = parts[i + 1]; i += 2; continue
    elif p.startswith("--file="):
        path = p.split("=", 1)[1]; i += 1; continue
    # combined short flags such as -am "msg"
    elif re.fullmatch(r"-[a-zA-Z]*m", p) and len(p) > 2:
        if i + 1 < len(parts):
            msgs.append(parts[i + 1]); i += 2; continue
    i += 1

if msgs:
    print("MSG\t" + "\n\n".join(msgs))
elif path:
    print("FILE\t" + path)
else:
    print("NOMSG")
PY
)"

KIND="${PARSED%%$'\t'*}"
VALUE="${PARSED#*$'\t'}"
[[ "$KIND" == "SKIP" ]] && exit 0

if [[ ! -f "$VALIDATOR" ]]; then
    echo "commit-convention: validator missing ($VALIDATOR) — follow $SKILL/SKILL.md by hand." >&2
    exit 0
fi

if [[ "$KIND" == "NOMSG" ]]; then
    echo "commit-convention: no -m message found (editor mode)." >&2
    echo "Read $SKILL/SKILL.md, then validate the file before finishing:" >&2
    echo "  bash $VALIDATOR --file .git/COMMIT_EDITMSG" >&2
    exit 0
fi

if [[ "$KIND" == "FILE" ]]; then
    [[ -f "$VALUE" ]] || exit 0
    OUT="$(bash "$VALIDATOR" --file "$VALUE" 2>&1)"; RC=$?
else
    OUT="$(bash "$VALIDATOR" "$VALUE" 2>&1)"; RC=$?
fi

[[ $RC -eq 0 ]] && exit 0

{
    echo "commit-convention: this commit message violates the team standard, so the commit was blocked."
    echo "$OUT"
    echo ""
    echo "Fix the message per $SKILL/SKILL.md — <type>(<scope>): <subject>, imperative, <= 72 chars,"
    echo "type in feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert — then commit again."
} >&2
exit 2
