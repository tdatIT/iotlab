#!/usr/bin/env bash
set -uo pipefail
# Stop — remind about the test gates when production Go code changed without a matching test change.
# Advisory only (always exit 0): the gate itself is `make coverage` / `make e2e`.
# Policy: .agents/skills/service-test/SKILL.md

cd "${CLAUDE_PROJECT_DIR:-.}" 2>/dev/null || exit 0
git rev-parse --is-inside-work-tree >/dev/null 2>&1 || exit 0

CHANGED="$(git status --porcelain -- 'internal/*' 'pkgs/*' 2>/dev/null | awk '{print $NF}' | grep -E '\.go$' || true)"
[[ -z "$CHANGED" ]] && exit 0

PROD="$(printf '%s\n' "$CHANGED" | grep -vE '_test\.go$|/mock/' || true)"
TESTS="$(printf '%s\n' "$CHANGED" | grep -E '_test\.go$' || true)"
[[ -z "$PROD" ]] && exit 0

if [[ -z "$TESTS" ]]; then
    echo "service-test: production Go code changed with no test change:"
    printf '%s\n' "$PROD" | sed 's/^/  /'
    echo "Definition of Done needs a test per branch. Gate: make coverage (>= 80%)."
fi
exit 0
