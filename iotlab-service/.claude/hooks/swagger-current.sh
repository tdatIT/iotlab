#!/usr/bin/env bash
set -uo pipefail
# PreToolUse(Bash) — before a `git commit`, refuse to ship an API contract the spec does not describe.
#
# Blocks (exit 2) when a new/changed route has no matching path+method in docs/swagger/swagger.json,
# or when the staged diff changes the router/handlers/DTOs without staging a regenerated spec.
# The rule it enforces: "regenerate Swagger in the same change" (swagger-api SKILL.md).
#
# `--no-verify` and `--amend` are left alone: the first is an explicit override, the second reworks
# a commit whose contract was already checked.

PAYLOAD="$(cat)"
ROOT="${CLAUDE_PROJECT_DIR:-.}"
CHECKER="$ROOT/.agents/skills/swagger-api/scripts/check-swagger.sh"
[[ -f "$CHECKER" ]] || exit 0

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
printf '%s' "$CMD" | grep -qE -- '--no-verify|--amend' && exit 0

cd "$ROOT" 2>/dev/null || exit 0
OUT="$(bash "$CHECKER" --staged 2>&1)"; RC=$?
[[ $RC -eq 0 ]] && exit 0
# A broken environment (missing spec, missing parser) must not block the commit.
[[ $RC -ne 1 ]] && exit 0

{
    echo "swagger-current: the OpenAPI spec does not match the routes, so the commit was blocked."
    echo "$OUT"
    echo "Fix it in this same commit: add the missing @Router annotations, run 'make swagger',"
    echo "then 'git add docs/swagger' and commit again."
} >&2
exit 2
