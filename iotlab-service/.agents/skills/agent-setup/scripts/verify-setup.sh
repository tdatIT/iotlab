#!/usr/bin/env bash
set -uo pipefail
# verify-setup.sh — Smoke-test the agent setup of this repo on this machine.
# Checks: repo root (anchored on go.mod) - instruction files - skills - tools - skill scripts -
# test gates - optional hooks. Reports PASS/WARN/FAIL, exits 1 on any FAIL.
# Read-only: touches nothing in the repo except temp files. bash 3.2 compatible (macOS).

VERSION="2.0.0"
usage() {
    cat <<'EOF'
verify-setup.sh — smoke-test the agent setup of this repo

  bash .agents/skills/agent-setup/scripts/verify-setup.sh [--quiet]

  --quiet      print the summary only
  -h, --help   this help
  --version    version
Exit: 0 = no FAIL - 1 = at least one FAIL (fix before developing)
EOF
}
QUIET=0
case "${1:-}" in
    -h|--help) usage; exit 0 ;;
    --version) echo "$VERSION"; exit 0 ;;
    --quiet) QUIET=1 ;;
esac

P=0; W=0; F=0
say()  { [[ "$QUIET" == 1 ]] || echo "$@"; }
ok()   { P=$((P+1)); say "  ok   $*"; }
warn() { W=$((W+1)); say "  warn $*"; }
fail() { F=$((F+1)); say "  FAIL $*"; }
have() { command -v "$1" >/dev/null 2>&1; }

# --- Repo root: the directory holding go.mod. Never anchor on agent config: ~/.claude/settings.json
# --- exists on most machines and would silently resolve the root to $HOME.
REPO=""; d="$PWD"
while [[ "$d" != "/" ]]; do
    [[ -f "$d/go.mod" ]] && { REPO="$d"; break; }
    d="$(dirname "$d")"
done

echo "-- verify-setup: agent setup --"

# 1) Repo root
say ""; say "1) Repo root"
if [[ -z "$REPO" ]]; then
    fail "no go.mod found walking up from $PWD — run this from inside the repo."
    echo ""; echo "Summary: PASS=$P WARN=$W FAIL=$F"; exit 1
fi
ok "repo root: $REPO"
[[ -d "$REPO/.git" ]] && ok ".git present" || warn ".git missing — not a clone?"

# 2) Instruction files
say ""; say "2) Instruction files"
for f in CLAUDE.md AGENTS.md docs/GUIDELINE.md; do
    [[ -f "$REPO/$f" ]] && ok "$f" || fail "$f missing"
done
WIRED="$(grep -oE '\.agents/skills/[a-z0-9-]+/SKILL\.md' "$REPO/CLAUDE.md" 2>/dev/null | sed -E 's#\.agents/skills/([a-z0-9-]+)/SKILL\.md#\1#' | sort -u)"
if [[ -n "$WIRED" ]]; then
    ok "CLAUDE.md always-load skills: $(echo "$WIRED" | tr '\n' ' ')"
else
    warn "CLAUDE.md wires in no skill — every skill loads only when named explicitly"
fi

# 3) Skills
say ""; say "3) Skills"
SKILL_DIRS="$(find "$REPO/.agents/skills" -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)"
if [[ -z "$SKILL_DIRS" ]]; then
    fail ".agents/skills is empty or missing"
else
    COUNT=0; BADNAME=0; DEADLINK=0
    for sd in $SKILL_DIRS; do
        base="$(basename "$sd")"
        if [[ ! -f "$sd/SKILL.md" ]]; then fail "$base: no SKILL.md"; continue; fi
        COUNT=$((COUNT+1))
        nm="$(grep -m1 '^name:' "$sd/SKILL.md" | sed -E 's/^name:[[:space:]]*//; s/[[:space:]]*$//')"
        [[ "$nm" == "$base" ]] || { warn "$base: frontmatter name is '$nm'"; BADNAME=$((BADNAME+1)); }
        # relative markdown links must resolve
        for lnk in $(grep -oE '\]\([^)#]+\.md[^)]*\)' "$sd/SKILL.md" | sed -E 's/^\]\(//; s/\)$//; s/#.*$//'); do
            case "$lnk" in http*|mailto:*) continue ;; esac
            [[ -e "$sd/$lnk" ]] || { warn "$base: dead link $lnk"; DEADLINK=$((DEADLINK+1)); }
        done
    done
    ok "$COUNT skills with a SKILL.md"
    [[ $BADNAME -eq 0 ]]  && ok "every frontmatter name matches its directory"
    [[ $DEADLINK -eq 0 ]] && ok "no dead links in any SKILL.md"
fi

# 4) Tools
say ""; say "4) Tools"
for t in bash git go; do have "$t" && ok "$t" || fail "$t (required)"; done
if have go && [[ -f "$REPO/go.mod" ]]; then
    WANT="$(grep -E '^go [0-9]' "$REPO/go.mod" | head -1 | awk '{print $2}')"
    GOT="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"
    [[ "${GOT%%.*}.${GOT#*.}" == "$WANT"* || "$GOT" == "$WANT"* ]] \
        && ok "go $GOT matches go.mod ($WANT)" \
        || warn "go $GOT vs go.mod $WANT — use test-docker.sh for CI-identical numbers"
fi
if have jq || have python3; then ok "jq/python3 (script JSON output)"; else warn "jq or python3 missing"; fi
for t in gofmt golangci-lint docker swag migrate; do
    have "$t" && ok "$t" || warn "$t missing — the workflow using it degrades (see references/SETUP.md)"
done

# 5) Skill scripts
say ""; say "5) Skill scripts"
SCRIPTS="$(find "$REPO/.agents/skills" -name '*.sh' -type f 2>/dev/null | sort)"
if [[ -z "$SCRIPTS" ]]; then
    warn "no skill scripts found"
else
    for sp in $SCRIPTS; do
        rel="${sp#$REPO/.agents/skills/}"
        if ! bash -n "$sp" 2>/dev/null; then fail "$rel: syntax error"; continue; fi
        bash "$sp" --help >/dev/null 2>&1 && ok "$rel --help" || warn "$rel: --help failed"
    done
fi

# 6) Test gates
say ""; say "6) Test gates"
for tgt in coverage e2e; do
    line="$(grep -A1 "^${tgt}:" "$REPO/Makefile" 2>/dev/null | tail -1)"
    gp="$(echo "$line" | grep -oE '\.agents/skills/[^ ]+\.sh')"
    if [[ -z "$gp" ]]; then warn "make $tgt: no skill script wired"
    elif [[ -f "$REPO/$gp" ]]; then ok "make $tgt -> $gp"
    else fail "make $tgt -> $gp (missing)"; fi
done
[[ -d "$REPO/test/e2e" ]] && ok "test/e2e suite present" || warn "test/e2e missing — Gate B has nothing to run"

# 7) Claude Code wiring: skills symlink + hooks (tracked in git, so absence means a broken checkout)
say ""; say "7) Claude Code wiring"
if [[ -d "$REPO/.claude" ]]; then
    ok ".claude present: $REPO/.claude"
    [[ -f "$REPO/.claude/settings.json" ]] && ok ".claude/settings.json" || fail ".claude exists without settings.json"
    if [[ -L "$REPO/.claude/skills" ]]; then
        [[ -d "$REPO/.claude/skills" ]] && ok ".claude/skills -> $(readlink "$REPO/.claude/skills") (skills discoverable)" \
                                        || fail ".claude/skills symlink is broken"
    else
        warn ".claude/skills missing — skills will not appear in Claude Code's skill list"
    fi
    HK="$REPO/.claude/hooks"
    if [[ -d "$HK" ]]; then
        for h in "$HK"/*.sh; do [[ -f "$h" ]] && { bash -n "$h" 2>/dev/null && ok "hook $(basename "$h"): syntax ok" || fail "hook $(basename "$h"): syntax error"; }; done
    else
        warn ".claude/hooks missing — settings.json may point at hooks that do not exist"
    fi
    say "  note hooks fire ONLY when Claude Code is opened at $REPO (the directory holding .claude/)."
else
    fail "no .claude/ — it is tracked in git; restore it with 'git checkout -- .claude' (see references/SETUP.md)"
fi

echo ""; echo "Summary: PASS=$P  WARN=$W  FAIL=$F"
[[ $F -eq 0 ]] && { echo "OK — setup usable."; exit 0; } || { echo "FAIL — fix the items above before developing."; exit 1; }
