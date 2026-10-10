#!/usr/bin/env bash
set -uo pipefail

# check-swagger.sh — Is the generated OpenAPI spec still telling the truth about the routes?
#
# S1 (--staged only) the commit changes the API contract (router / handler / DTOs) but stages no
#    regenerated spec — the rule in SKILL.md is "regenerate Swagger in the same change".
# S2 a business route in internal/router/routes.go has no matching path+method in
#    docs/swagger/swagger.json — a missing @Router annotation, or docs never regenerated.
#
# The route list comes from service-test's list-apis.sh, the repo's single route parser.

VERSION="1.0.0"
SCRIPT_NAME="$(basename "$0")"
SPEC="docs/swagger/swagger.json"
LIST_APIS=".agents/skills/service-test/scripts/list-apis.sh"
# Files whose change alters the published contract.
CONTRACT_RE='^internal/router/|^internal/handler/|^internal/domain/dtos/'

usage() {
    cat <<EOF
$SCRIPT_NAME v$VERSION — Check the OpenAPI spec against the actual routes

USAGE
    bash $SCRIPT_NAME [options]

DESCRIPTION
    S2 (always): every business route in internal/router/routes.go must appear in $SPEC
       with the same method. Route paths map to spec paths by stripping basePath and
       rewriting ':id' to '{id}'.
    S1 (--staged): when the staged diff touches the router, a handler or a DTO, the
       regenerated spec must be staged in the same commit.

    Regenerate with: make swagger

OPTIONS
    -h, --help      Show this help
    -v, --version   Show version
    --staged        Also run S1 against the staged diff (used by the pre-commit hook)
    --json          Machine-readable summary

    Exit 0 = spec is current, 1 = out of date, 2 = bad arguments or environment.
EOF
}

STAGED=false
JSON=false
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)    usage; exit 0 ;;
        -v|--version) echo "$SCRIPT_NAME v$VERSION"; exit 0 ;;
        --staged)     STAGED=true; shift ;;
        --json)       JSON=true; shift ;;
        -*)           echo "error: unknown option: $1" >&2; usage >&2; exit 2 ;;
        *)            echo "error: unexpected arg: $1" >&2; exit 2 ;;
    esac
done

command -v python3 >/dev/null 2>&1 || { echo "error: python3 required" >&2; exit 2; }
[[ -f go.mod ]] || { echo "error: run from the service root" >&2; exit 2; }
[[ -f "$LIST_APIS" ]] || { echo "error: route parser missing: $LIST_APIS" >&2; exit 2; }
[[ -f "$SPEC" ]] || { echo "error: spec missing: $SPEC — run 'make swagger'" >&2; exit 2; }

ROUTES_JSON="$(bash "$LIST_APIS" --json 2>/dev/null)"
[[ -z "$ROUTES_JSON" ]] && { echo "error: could not read the route list" >&2; exit 2; }

STAGED_FILES=""
if $STAGED; then
    STAGED_FILES="$(git diff --cached --name-only --diff-filter=ACMR 2>/dev/null || true)"
fi

python3 - "$SPEC" "$STAGED" "$JSON" "$CONTRACT_RE" <<PY
import json, re, sys

spec_path, staged_flag, json_flag, contract_re = sys.argv[1:5]
staged = staged_flag == "true"
as_json = json_flag == "true"
routes = json.loads(r'''$ROUTES_JSON''')["routes"]
staged_files = [l for l in r'''$STAGED_FILES'''.split("\n") if l.strip()]

spec = json.load(open(spec_path, encoding="utf-8"))
base = (spec.get("basePath") or "").rstrip("/")
spec_paths = spec.get("paths", {})

def to_spec_path(p):
    if base and p.startswith(base):
        p = p[len(base):] or "/"
    return re.sub(r":([A-Za-z_]\w*)", r"{\1}", p)

missing = []
for r in routes:
    if r["platform"]:
        continue
    sp = to_spec_path(r["path"])
    methods = {m.lower() for m in spec_paths.get(sp, {}) if m.lower() in
               ("get", "post", "put", "patch", "delete", "head", "options")}
    if r["method"].lower() not in methods:
        missing.append({"method": r["method"], "path": r["path"], "spec_path": sp,
                        "handler": r["handler"]})

contract_changed = [f for f in staged_files if re.search(contract_re, f)] if staged else []
spec_staged = any(f.startswith("docs/swagger/") for f in staged_files)
stale = bool(contract_changed) and not spec_staged

if as_json:
    print(json.dumps({"missing": missing, "contract_changed": contract_changed,
                      "spec_staged": spec_staged, "stale": stale,
                      "pass": not missing and not stale}))
    sys.exit(0 if (not missing and not stale) else 1)

if stale:
    print("S1 the staged diff changes the API contract but no regenerated spec is staged:")
    for f in contract_changed:
        print(f"  {f}")
    print("  -> make swagger && git add docs/swagger")
    print()

if missing:
    print("S2 routes with no matching path+method in the spec:")
    for m in missing:
        print(f"  {m['method']:<7} {m['path']}  (expected \"{m['spec_path']}\" in {spec_path}) -> {m['handler']}")
    print("  -> add the @Router annotation above the handler, then: make swagger")
    print()

if not missing and not stale:
    print(f"swagger OK ({len([r for r in routes if not r['platform']])} business route(s) documented)")
    sys.exit(0)

print("Rules: .agents/skills/swagger-api/SKILL.md")
sys.exit(1)
PY
