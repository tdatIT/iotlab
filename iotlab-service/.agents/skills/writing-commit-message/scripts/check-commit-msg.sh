#!/usr/bin/env bash
set -euo pipefail

# check-commit-msg.sh — Validate a commit message against Conventional Commits + team rules.
# Run directly or install as a git commit-msg hook.

VERSION="1.1.0"
SCRIPT_NAME="$(basename "$0")"

usage() {
    cat <<HELP
$SCRIPT_NAME v$VERSION — Validate commit message (Conventional Commits)

USAGE
    bash $SCRIPT_NAME "feat(user): add create endpoint"
    bash $SCRIPT_NAME --file .git/COMMIT_EDITMSG
    or install as .git/hooks/commit-msg

OPTIONS
    -h, --help     Show help
    -v, --version  Show version
    --file PATH    Read message from file (for the git commit-msg hook)

RULES
    - Header: <type>(<scope>)?: <subject>
    - type in feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert
    - subject: <= 72 chars, no trailing period, imperative mood
    - Line 2 must be empty when a body follows

    Exit 0 = valid, 1 = rule violation, 2 = bad arguments.
HELP
}

MSG=""
FROM_FILE=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)    usage; exit 0 ;;
        -v|--version) echo "$SCRIPT_NAME v$VERSION"; exit 0 ;;
        --file)       FROM_FILE="${2:?error: --file requires a path}"; shift 2 ;;
        -*)           echo "error: unknown option: $1" >&2; exit 2 ;;
        *)            MSG="$1"; shift ;;
    esac
done

if [[ -n "$FROM_FILE" ]]; then
    [[ -f "$FROM_FILE" ]] || { echo "error: file not found: $FROM_FILE" >&2; exit 2; }
    MSG="$(cat "$FROM_FILE")"
fi
[[ -z "$MSG" ]] && { echo "error: missing commit message" >&2; usage >&2; exit 2; }

# Strip comments (#...) and leading blank lines when reading from the editor file
SUBJECT="$(printf '%s\n' "$MSG" | grep -vE '^\s*#' | sed '/./,$!d' | head -n1)"
SECOND_LINE="$(printf '%s\n' "$MSG" | grep -vE '^\s*#' | sed -n '2p')"

ERRORS=()

TYPES='feat|fix|docs|style|refactor|perf|test|build|ci|chore|revert'
HEADER_RE="^(${TYPES})(\([a-z0-9._/-]+\))?(!)?: .+"

if ! printf '%s' "$SUBJECT" | grep -qE "$HEADER_RE"; then
    ERRORS+=("Bad header format. Expected: <type>(<scope>): <subject>  (type in ${TYPES})")
fi

SUBJECT_LEN=${#SUBJECT}
if [[ $SUBJECT_LEN -gt 72 ]]; then
    ERRORS+=("Header is $SUBJECT_LEN chars (>72). Shorten it.")
fi

if printf '%s' "$SUBJECT" | grep -qE '\.$'; then
    ERRORS+=("Header must not end with a period.")
fi

# text after "type(scope): "
DESC="$(printf '%s' "$SUBJECT" | sed -E "s/^(${TYPES})(\([^)]*\))?(!)?: //")"
if printf '%s' "$DESC" | grep -qE '^(added|fixed|updated|changed|removed|adds|fixes)\b'; then
    ERRORS+=("Use imperative mood: 'add' instead of 'added'/'adds'.")
fi

if [[ -n "$SECOND_LINE" ]]; then
    ERRORS+=("Line 2 must be empty (separates header from body).")
fi

if [[ ${#ERRORS[@]} -eq 0 ]]; then
    echo "✅ Valid commit message: $SUBJECT"
    exit 0
fi

echo "🚨 Invalid commit message:"
printf '   - %s\n' "${ERRORS[@]}"
echo ""
echo "Example: feat(user): add paginated list endpoint"
exit 1
