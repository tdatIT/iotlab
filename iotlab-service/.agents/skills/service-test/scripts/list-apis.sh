#!/usr/bin/env bash
set -euo pipefail

# list-apis.sh — Rule 4: list every HTTP API the service exposes, which ones are selected for
# e2e, and which ones already have an e2e test. Also enforces Gate B (e2e API coverage).
#
# Sources of truth:
#   internal/router/routes.go   — the API list (parsed, always current)
#   test/e2e/*_test.go          — '// @e2e METHOD PATH' annotations declare what a test covers
#   test/e2e/.e2e-apis          — the selected subset; absent means "all business APIs selected"

VERSION="1.0.0"
SCRIPT_NAME="$(basename "$0")"

ROUTES_FILE="internal/router/routes.go"
E2E_DIR="test/e2e"
SELECT_FILE="$E2E_DIR/.e2e-apis"

usage() {
    cat <<EOF
$SCRIPT_NAME v$VERSION — List the service APIs and their e2e status

USAGE
    bash $SCRIPT_NAME [options]

DESCRIPTION
    Parses $ROUTES_FILE and prints every route with three flags:
      PLATFORM  infrastructure route (/metrics, /liveness, /readiness, /swagger*) — never auto-selected
      SELECTED  in the e2e scope (all business APIs by default, or whatever $SELECT_FILE lists)
      E2E       an e2e test declares '// @e2e METHOD PATH' for it

    Show this list to the user and let them pick the e2e scope. The DEFAULT scope is
    EVERY business API. Write the chosen subset to $SELECT_FILE (one 'METHOD PATH' per line).

    --gate turns it into Gate B: fails when a selected API has no e2e test.

OPTIONS
    -h, --help          Show this help
    -v, --version       Show version
    --json              Machine-readable output (for the agent)
    --gate              Exit 1 when e2e API coverage is below the threshold
    --threshold N       Gate B threshold as a percentage (default: 100)
    --routes FILE       Override the routes file (default: $ROUTES_FILE)

EXAMPLES
    bash $SCRIPT_NAME                       # show the table, pick a scope with the user
    bash $SCRIPT_NAME --json                # same data, machine-readable
    bash $SCRIPT_NAME --gate                # Gate B: every selected API must have an e2e test
EOF
}

JSON=false
GATE=false
THRESHOLD=100

while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)    usage; exit 0 ;;
        -v|--version) echo "$SCRIPT_NAME v$VERSION"; exit 0 ;;
        --json)       JSON=true; shift ;;
        --gate)       GATE=true; shift ;;
        --threshold)  THRESHOLD="${2:?}"; shift 2 ;;
        --routes)     ROUTES_FILE="${2:?}"; shift 2 ;;
        -*)           echo "error: unknown option: $1" >&2; usage >&2; exit 2 ;;
        *)            echo "error: unexpected arg: $1" >&2; exit 2 ;;
    esac
done

[[ -f "$ROUTES_FILE" ]] || { echo "error: routes file not found: $ROUTES_FILE (run from the service root)" >&2; exit 2; }

# --- 1. Routes: resolve echo Group chains, then every e.METHOD / group.METHOD registration ---
ROUTES="$(awk '
    # group var := parent.Group("/prefix")
    match($0, /^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*[[:space:]]*:=[[:space:]]*[A-Za-z_][A-Za-z0-9_]*\.Group\("[^"]*"\)/) {
        line = $0
        sub(/^[[:space:]]*/, "", line)
        split(line, a, /[[:space:]]*:=[[:space:]]*/)
        child = a[1]
        rest = a[2]
        split(rest, b, /\.Group\("/)
        parent = b[1]
        lit = b[2]
        sub(/".*$/, "", lit)
        prefix[child] = prefix[parent] lit
        next
    }
    # var.METHOD("/path", handler)
    match($0, /[A-Za-z_][A-Za-z0-9_]*\.(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\("/) {
        seg = substr($0, RSTART, RLENGTH)
        split(seg, c, /\./)
        v = c[1]; m = c[2]
        sub(/\("$/, "", m)
        rest = substr($0, RSTART + RLENGTH)
        lit = rest
        sub(/".*$/, "", lit)
        # handler: text after the first comma, keep the final identifier segment
        h = rest
        sub(/^[^,]*,[[:space:]]*/, "", h)
        if (h ~ /^func/) { h = "inline" }
        else {
            sub(/\).*$/, "", h)
            n = split(h, d, /\./)
            h = d[n]
            sub(/\(.*$/, "", h)
        }
        path = prefix[v] lit
        if (path == "") path = "/"
        printf "%s\t%s\t%s\n", m, path, h
    }
' "$ROUTES_FILE")"

[[ -n "$ROUTES" ]] || { echo "error: no routes parsed from $ROUTES_FILE" >&2; exit 2; }

# --- 2. e2e annotations: '// @e2e METHOD PATH' ---
COVERED=""
if [[ -d "$E2E_DIR" ]]; then
    COVERED="$(grep -rhoE '@e2e[[:space:]]+(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)[[:space:]]+[^[:space:]]+' "$E2E_DIR" 2>/dev/null \
        | awk '{printf "%s\t%s\n", $2, $3}' | sort -u || true)"
fi

# --- 3. Selection: .e2e-apis, or every business API by default ---
SELECTED_LIST=""
SELECT_MODE="default (all business APIs)"
if [[ -f "$SELECT_FILE" ]]; then
    SELECT_MODE="$SELECT_FILE"
    SELECTED_LIST="$(grep -vE '^[[:space:]]*(#|$)' "$SELECT_FILE" | awk '{printf "%s\t%s\n", $1, $2}' | sort -u || true)"
fi

# --- 4. Join everything into: METHOD PATH HANDLER PLATFORM SELECTED E2E ---
TABLE="$(printf '%s\n' "$ROUTES" | awk -v covered="$COVERED" -v selected="$SELECTED_LIST" -v mode="$SELECT_MODE" '
    BEGIN {
        FS = OFS = "\t"
        n = split(covered, cl, "\n"); for (i = 1; i <= n; i++) if (cl[i] != "") cov[cl[i]] = 1
        m = split(selected, sl, "\n"); for (i = 1; i <= m; i++) if (sl[i] != "") sel[sl[i]] = 1
        useFile = (mode != "default (all business APIs)")
    }
    {
        key = $1 OFS $2
        platform = ($2 !~ /^\/api\//) ? "yes" : "no"
        if (useFile) isSel = (key in sel) ? "yes" : "no"
        else         isSel = (platform == "yes") ? "no" : "yes"
        hasE2E = (key in cov) ? "yes" : "no"
        print $1, $2, $3, platform, isSel, hasE2E
    }
')"

SEL_TOTAL="$(printf '%s\n' "$TABLE" | awk -F'\t' '$5=="yes"' | grep -c . || true)"
SEL_DONE="$(printf '%s\n' "$TABLE" | awk -F'\t' '$5=="yes" && $6=="yes"' | grep -c . || true)"
PCT="$(awk -v d="${SEL_DONE:-0}" -v t="${SEL_TOTAL:-0}" 'BEGIN{ if (t+0==0) print "100.0"; else printf "%.1f", (d*100.0)/t }')"
PASS="$(awk -v a="$PCT" -v b="$THRESHOLD" 'BEGIN{print (a+0>=b+0)?"yes":"no"}')"

if $JSON; then
    printf '{"selection_source":"%s","selected":%s,"with_e2e":%s,"api_coverage":%s,"threshold":%s,"pass":%s,"routes":[' \
        "$SELECT_MODE" "${SEL_TOTAL:-0}" "${SEL_DONE:-0}" "$PCT" "$THRESHOLD" \
        "$([[ "$PASS" == yes ]] && echo true || echo false)"
    printf '%s\n' "$TABLE" | awk -F'\t' '
        NR>1 { printf "," }
        { printf "{\"method\":\"%s\",\"path\":\"%s\",\"handler\":\"%s\",\"platform\":%s,\"selected\":%s,\"e2e\":%s}",
                 $1, $2, $3, ($4=="yes"?"true":"false"), ($5=="yes"?"true":"false"), ($6=="yes"?"true":"false") }'
    printf ']}\n'
else
    echo ""
    printf '%-3s %-7s %-26s %-16s %-9s %-9s %s\n' "#" "METHOD" "PATH" "HANDLER" "PLATFORM" "SELECTED" "E2E"
    printf '%s\n' "$TABLE" | awk -F'\t' '
        { printf "%-3d %-7s %-26s %-16s %-9s %-9s %s\n", NR, $1, $2, $3, ($4=="yes"?"platform":"-"), ($5=="yes"?"yes":"-"), ($5=="yes" ? ($6=="yes"?"yes":"MISSING") : "-") }'
    echo ""
    echo "Selection source: $SELECT_MODE"
    echo "e2e API coverage: ${SEL_DONE:-0}/${SEL_TOTAL:-0} selected (${PCT}%)  | threshold: ${THRESHOLD}%"
    MISSING="$(printf '%s\n' "$TABLE" | awk -F'\t' '$5=="yes" && $6=="no" { printf "  %s %s\n", $1, $2 }')"
    if [[ -n "$MISSING" ]]; then
        echo ""
        echo "Selected APIs with NO e2e test — add '// @e2e METHOD PATH' above a test in $E2E_DIR:"
        printf '%s\n' "$MISSING"
    fi
fi

if $GATE; then
    if [[ "$PASS" == yes ]]; then
        $JSON || echo "PASS: e2e API coverage threshold met."
        exit 0
    fi
    $JSON || echo "FAIL: e2e API coverage below threshold — not Done yet."
    exit 1
fi
exit 0
