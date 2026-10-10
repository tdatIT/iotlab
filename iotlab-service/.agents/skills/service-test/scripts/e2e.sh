#!/usr/bin/env bash
set -euo pipefail

# e2e.sh — Gate B: run the e2e suite against real dependencies, then enforce e2e API coverage
# over the selected API list (default: every business API).
#
# The suite lives in test/e2e, is guarded by the 'e2e' build tag, and declares what each test
# covers with '// @e2e METHOD PATH'. list-apis.sh turns those annotations into the gate.

VERSION="1.0.0"
SCRIPT_NAME="$(basename "$0")"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

E2E_DIR="./test/e2e/..."
COMPOSE_DEPS=(postgres emqx5)

usage() {
    cat <<EOF
$SCRIPT_NAME v$VERSION — Gate B: run the e2e suite and gate e2e API coverage

USAGE
    bash $SCRIPT_NAME [options]

DESCRIPTION
    1. Brings up postgres/emqx5 via docker compose (unless --no-deps).
    2. Runs 'go test -tags e2e $E2E_DIR'.
    3. Runs list-apis.sh --gate: every SELECTED API must have an e2e test.

    Gate A — unit line coverage — is enforced by coverage.sh, not here.

    Exit 0 = tests pass and the API-coverage threshold is met, 1 = failure, 2 = bad argument
    or environment.

OPTIONS
    -h, --help          Show this help
    -v, --version       Show version
    --no-deps           Assume the dependencies are already running
    --keep-deps         Leave the compose stack up afterwards
    --threshold N       e2e API coverage threshold (default: 100)
    --run REGEX         Passed through to 'go test -run'
    --skip-gate         Run the tests only, skip the API-coverage gate
    --print             Print the commands without running them

REQUIREMENTS
    go, docker + docker compose (unless --no-deps), run from the service root.

EXAMPLES
    bash $SCRIPT_NAME                       # deps up, run e2e, gate at 100%
    bash $SCRIPT_NAME --no-deps --run Create
    bash $SCRIPT_NAME --threshold 60        # temporarily accept partial e2e coverage
EOF
}

WITH_DEPS=true
KEEP_DEPS=false
THRESHOLD=100
RUN_REGEX=""
SKIP_GATE=false
PRINT=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)    usage; exit 0 ;;
        -v|--version) echo "$SCRIPT_NAME v$VERSION"; exit 0 ;;
        --no-deps)    WITH_DEPS=false; shift ;;
        --keep-deps)  KEEP_DEPS=true; shift ;;
        --threshold)  THRESHOLD="${2:?}"; shift 2 ;;
        --run)        RUN_REGEX="${2:?}"; shift 2 ;;
        --skip-gate)  SKIP_GATE=true; shift ;;
        --print)      PRINT=true; shift ;;
        -*)           echo "error: unknown option: $1" >&2; usage >&2; exit 2 ;;
        *)            echo "error: unexpected arg: $1" >&2; exit 2 ;;
    esac
done

command -v go >/dev/null 2>&1 || { echo "error: Go toolchain required" >&2; exit 2; }
[[ -f go.mod ]] || { echo "error: go.mod not found (run from the service root)" >&2; exit 2; }
[[ -d test/e2e ]] || { echo "error: test/e2e not found — create the e2e suite first" >&2; exit 2; }

# The e2e harness talks to the compose stack; config.yaml defaults point at localhost, and the
# compose postgres password is not in the repo config, so it is supplied here.
export DATABASE_HOST="${DATABASE_HOST:-localhost}"
export DATABASE_PASSWORD="${DATABASE_PASSWORD:-devpass@123}"
export REDIS_ADDRESS="${REDIS_ADDRESS:-127.0.0.1:6379}"
export MQTT_BROKER="${MQTT_BROKER:-tcp://localhost:1883}"

GO_TEST=(go test -tags e2e "$E2E_DIR" -count=1 -v)
[[ -n "$RUN_REGEX" ]] && GO_TEST+=(-run "$RUN_REGEX")

if $PRINT; then
    echo "# threshold: ${THRESHOLD}% | with-deps: $WITH_DEPS"
    $WITH_DEPS && echo "docker compose up -d ${COMPOSE_DEPS[*]}"
    printf '%q ' "${GO_TEST[@]}"; echo
    $SKIP_GATE || echo "bash $SCRIPT_DIR/list-apis.sh --gate --threshold $THRESHOLD"
    exit 0
fi

if $WITH_DEPS; then
    command -v docker >/dev/null 2>&1 || { echo "error: docker not installed (use --no-deps)" >&2; exit 2; }
    echo "> docker compose up -d ${COMPOSE_DEPS[*]}"
    docker compose up -d --wait "${COMPOSE_DEPS[@]}" 2>/dev/null || docker compose up -d "${COMPOSE_DEPS[@]}"
fi

echo "> ${GO_TEST[*]}"
set +e
"${GO_TEST[@]}"
RC=$?
set -e

if $WITH_DEPS && ! $KEEP_DEPS; then
    echo "> docker compose down"
    docker compose down >/dev/null 2>&1 || true
fi

[[ $RC -ne 0 ]] && { echo "FAIL: e2e tests failed." >&2; exit 1; }

if ! $SKIP_GATE; then
    echo ""
    bash "$SCRIPT_DIR/list-apis.sh" --gate --threshold "$THRESHOLD" || exit 1
fi
exit 0
