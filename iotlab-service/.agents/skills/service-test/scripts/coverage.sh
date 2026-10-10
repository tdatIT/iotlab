#!/usr/bin/env bash
set -euo pipefail

# coverage.sh — Gate A: run unit tests and enforce a minimum line-coverage threshold (default 80%).
# Generated / mock / cmd / e2e files are dropped from the denominator. Untested (0%) functions are listed.

VERSION="2.0.0"
SCRIPT_NAME="$(basename "$0")"

usage() {
    cat <<EOF
$SCRIPT_NAME v$VERSION — Gate A: minimum unit-test line coverage for a Go service

USAGE
    bash $SCRIPT_NAME [options] [packages]

DESCRIPTION
    Runs 'go test -coverprofile' then computes total coverage. FAILS below the threshold.
    Excluded from the denominator by default: */mock/*, *_mock.go, *.pb.go, *_gen.go,
    *.gen.go, docs/swagger/, cmd/ (entrypoint), test/e2e/ (Gate B owns e2e).
    Untested (0%) functions are listed so you can see what is still missing.

    Gate B — e2e coverage of the selected API list — is enforced by e2e.sh, not here.

    Exit 0 = threshold met, 1 = below threshold / test failure, 2 = bad argument or environment.

OPTIONS
    -h, --help          Show this help
    -v, --version       Show version
    --threshold N       Minimum coverage percentage (default: 80)
    --exclude REGEX     Extra exclude pattern (ERE, appended to the defaults)
    --profile FILE      Coverage profile filename (default: coverage.out)
    --no-run            Skip 'go test', reuse an existing profile
    --json              Print a JSON summary

ARGUMENTS
    packages            Packages to test (default: ./...)

EXAMPLES
    bash $SCRIPT_NAME
    bash $SCRIPT_NAME --threshold 80
    bash $SCRIPT_NAME --exclude '/testutil/' ./internal/...
EOF
}

THRESHOLD=80
PROFILE="coverage.out"
RUN=true
JSON=false
PKGS="./..."
EXCLUDE_EXTRA=""

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)    usage; exit 0 ;;
        -v|--version) echo "$SCRIPT_NAME v$VERSION"; exit 0 ;;
        --threshold)  THRESHOLD="${2:?}"; shift 2 ;;
        --exclude)    EXCLUDE_EXTRA="${EXCLUDE_EXTRA}|${2:?}"; shift 2 ;;
        --profile)    PROFILE="${2:?}"; shift 2 ;;
        --no-run)     RUN=false; shift ;;
        --json)       JSON=true; shift ;;
        -*)           echo "error: unknown option: $1" >&2; usage >&2; exit 2 ;;
        *)            PKGS="$1"; shift ;;
    esac
done

command -v go >/dev/null 2>&1 || { echo "error: Go toolchain required" >&2; exit 2; }
[[ -f go.mod ]] || { echo "error: go.mod not found (run from the service root)" >&2; exit 2; }

# Excluded from the coverage denominator:
#  - generated code (mocks, protobuf, swagger docs) — nobody hand-writes it
#  - /cmd/ — entrypoint, wiring only
#  - /test/e2e/ — e2e suite, gated by e2e.sh on API coverage instead
# Bootstrap wiring and infrastructure connectors are NOT excluded: their error branches are
# unit-testable (bad config, unreachable DSN, dead broker) and excluding them empties the gate.
EXCLUDE='/mock/|_mock\.go|\.pb\.go|_gen\.go|\.gen\.go|/docs/swagger/|/cmd/|/test/e2e/'
[[ -n "$EXCLUDE_EXTRA" ]] && EXCLUDE="${EXCLUDE}${EXCLUDE_EXTRA}"

if $RUN; then
    echo "> go test -coverprofile ($PKGS) ..."
    if ! go test "$PKGS" -coverprofile="$PROFILE" -covermode=atomic -coverpkg=./... >/dev/null 2>test_err.log; then
        echo "FAIL: go test failed — fix the tests before measuring coverage:" >&2
        cat test_err.log >&2 || true
        rm -f test_err.log
        exit 1
    fi
    rm -f test_err.log
fi
[[ -f "$PROFILE" ]] || { echo "error: profile not found: $PROFILE" >&2; exit 2; }

# Drop excluded files from the profile (keep the 'mode:' header line)
FILTERED="${PROFILE}.filtered"
{ head -n1 "$PROFILE"; tail -n +2 "$PROFILE" | grep -vE "$EXCLUDE" || true; } > "$FILTERED"

FUNC_OUT="$(go tool cover -func="$FILTERED" 2>/dev/null || true)"
TOTAL_LINE="$(printf '%s\n' "$FUNC_OUT" | grep -E '^total:' | tail -1)"
TOTAL_PCT="$(printf '%s' "$TOTAL_LINE" | grep -oE '[0-9]+\.[0-9]+' | tail -1)"
[[ -z "$TOTAL_PCT" ]] && TOTAL_PCT="0.0"

# Untested functions (0%) — excluding the total line
UNTESTED="$(printf '%s\n' "$FUNC_OUT" | grep -vE '^total:' | awk '$NF=="0.0%"{print}')"
UNTESTED_COUNT="$(printf '%s' "$UNTESTED" | grep -c . || true)"

# Float compare via awk (bash has no float arithmetic)
PASS="$(awk -v a="$TOTAL_PCT" -v b="$THRESHOLD" 'BEGIN{print (a+0>=b+0)?"yes":"no"}')"

if $JSON; then
    printf '{"total_coverage":%s,"threshold":%s,"pass":%s,"untested_functions":%s}\n' \
        "$TOTAL_PCT" "$THRESHOLD" "$([[ "$PASS" == yes ]] && echo true || echo false)" "${UNTESTED_COUNT:-0}"
else
    echo ""
    echo "Total coverage (generated/mock/cmd/e2e excluded): ${TOTAL_PCT}%  | threshold: ${THRESHOLD}%"
    if [[ "${UNTESTED_COUNT:-0}" -gt 0 ]]; then
        echo ""
        echo "Functions with NO test (0%) — write tests for the APIs/handlers listed here first:"
        printf '%s\n' "$UNTESTED" | sed 's/^/  /'
    fi
    echo ""
    if [[ "$PASS" == yes ]]; then
        echo "PASS: coverage threshold met."
    else
        echo "FAIL: below threshold — not Done yet. Add tests for the 0% entries above."
    fi
fi

rm -f "$FILTERED"
[[ "$PASS" == yes ]] && exit 0 || exit 1
