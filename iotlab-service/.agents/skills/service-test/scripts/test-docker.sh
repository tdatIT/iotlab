#!/usr/bin/env bash
set -euo pipefail

# test-docker.sh — Run the gates INSIDE Docker, pinned to the go.mod toolchain so local results
# match whatever image CI would use. This is a thin wrapper: the gate logic lives in
# coverage.sh (Gate A) and e2e.sh (Gate B) and is not duplicated here.
#
# Default:  Gate A only (unit coverage) — no dependencies needed.
# --e2e:    Gate B as well — brings up postgres/emqx5 and joins the container to that network.

VERSION="2.0.0"
SCRIPT_NAME="$(basename "$0")"
SKILL_SCRIPTS=".agents/skills/service-test/scripts"
COMPOSE_DEPS=(postgres emqx5)

usage() {
    cat <<EOF
$SCRIPT_NAME v$VERSION — Run the test gates in Docker

USAGE
    bash $SCRIPT_NAME [options]

DESCRIPTION
    Runs 'go build ./...' then Gate A (coverage.sh) inside golang:<go.mod version>-alpine.
    With --e2e it also starts the compose dependencies and runs Gate B (e2e.sh) in the
    same container, attached to the compose network.

OPTIONS
    -h, --help          Show this help
    -v, --version       Show version
    --threshold N       Gate A coverage threshold (default: 80)
    --e2e               Also run Gate B (needs docker compose)
    --e2e-threshold N   Gate B e2e API coverage threshold (default: 100)
    --image NAME        Override the image (default: golang:<go.mod version>-alpine)
    --network NAME      Compose network name for --e2e (default: <directory>_default)
    --keep-deps         Do not 'compose down' after --e2e
    --print             Print the docker command without running it

REQUIREMENTS
    docker (plus docker compose for --e2e), go.mod in the current directory.

EXAMPLES
    bash $SCRIPT_NAME                       # Gate A in docker
    bash $SCRIPT_NAME --threshold 80
    bash $SCRIPT_NAME --e2e                 # Gate A + Gate B with real dependencies
    bash $SCRIPT_NAME --print
EOF
}

THRESHOLD=80
E2E_THRESHOLD=100
IMAGE=""
WITH_E2E=false
NETWORK=""
KEEP_DEPS=false
PRINT=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)        usage; exit 0 ;;
        -v|--version)     echo "$SCRIPT_NAME v$VERSION"; exit 0 ;;
        --threshold)      THRESHOLD="${2:?}"; shift 2 ;;
        --e2e)            WITH_E2E=true; shift ;;
        --e2e-threshold)  E2E_THRESHOLD="${2:?}"; shift 2 ;;
        --image)          IMAGE="${2:?}"; shift 2 ;;
        --network)        NETWORK="${2:?}"; shift 2 ;;
        --keep-deps)      KEEP_DEPS=true; shift ;;
        --print)          PRINT=true; shift ;;
        -*)               echo "error: unknown option: $1" >&2; usage >&2; exit 2 ;;
        *)                echo "error: unexpected arg: $1" >&2; exit 2 ;;
    esac
done

[[ -f go.mod ]] || { echo "error: go.mod not found (run from the service root)" >&2; exit 2; }

# Toolchain version from go.mod (the 'go 1.26.2' line)
GO_VER="$(grep -E '^go [0-9]' go.mod | head -1 | awk '{print $2}')"
[[ -z "$GO_VER" ]] && GO_VER="1.26"
[[ -z "$IMAGE" ]] && IMAGE="golang:${GO_VER}-alpine"

ENV_ARGS=()
if $WITH_E2E; then
    [[ -z "$NETWORK" ]] && NETWORK="$(basename "$PWD" | tr '[:upper:]' '[:lower:]')_default"
    # Inside the compose network the dependencies are reachable by service name.
    ENV_ARGS=(-e DATABASE_HOST=postgres -e MQTT_BROKER=tcp://emqx5:1883)
fi

# Commands run inside the container (sh, not bash — but the gate scripts need bash, so install it).
INNER="set -e
apk add --no-cache bash >/dev/null
echo '> go build ./...' && go build ./...
bash $SKILL_SCRIPTS/coverage.sh --threshold ${THRESHOLD}"
if $WITH_E2E; then
    INNER="$INNER
bash $SKILL_SCRIPTS/e2e.sh --no-deps --threshold ${E2E_THRESHOLD}"
fi

DOCKER_RUN=(docker run --rm
    -v "$PWD":/src -w /src
    -v vm-gomod-cache:/go/pkg/mod
    -e GOFLAGS=-buildvcs=false)
if $WITH_E2E; then
    DOCKER_RUN+=(--network "$NETWORK" "${ENV_ARGS[@]}")
fi
DOCKER_RUN+=("$IMAGE" sh -c "$INNER")

if $PRINT; then
    echo "# image: $IMAGE | gate A threshold: ${THRESHOLD}% | e2e: $WITH_E2E"
    $WITH_E2E && echo "docker compose up -d ${COMPOSE_DEPS[*]}   # network: $NETWORK"
    printf '%q ' "${DOCKER_RUN[@]}"; echo
    exit 0
fi

command -v docker >/dev/null 2>&1 || { echo "error: docker not installed" >&2; exit 2; }

if $WITH_E2E; then
    echo "> docker compose up -d ${COMPOSE_DEPS[*]}"
    docker compose up -d --wait "${COMPOSE_DEPS[@]}" 2>/dev/null || docker compose up -d "${COMPOSE_DEPS[@]}"
fi

set +e
"${DOCKER_RUN[@]}"
RC=$?
set -e

if $WITH_E2E && ! $KEEP_DEPS; then
    echo "> docker compose down"
    docker compose down >/dev/null 2>&1 || true
fi

exit $RC
