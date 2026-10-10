#!/usr/bin/env bash
set -uo pipefail

# check-comments.sh — Check Go comments against the rules in go-style-core (SKILL.md, "Comments").
#
# C1  a comment line must be <= 130 characters
# C2  at most 2 consecutive comment lines inside a function body — longer explanations belong at
#     the top of the file, right after the package declaration
# C3  a doc comment on an exported identifier must start with that identifier's name (Go convention)
#
# Skipped: generated files, vendor/, and C3 in *_test.go (test functions are not API docs).
# swag annotations (`@Param`, `@Success`, …) and `//go:` directives are exempt from C1: their
# layout is fixed by the tool that reads them.

VERSION="1.0.0"
SCRIPT_NAME="$(basename "$0")"
MAXLEN=130

usage() {
    cat <<EOF
$SCRIPT_NAME v$VERSION — Check Go comments against go-style-core

USAGE
    bash $SCRIPT_NAME [options] [files...]

DESCRIPTION
    Checks the comment rules of .agents/skills/go-style-core/SKILL.md:
      C1  comment line longer than $MAXLEN characters
      C2  more than 2 consecutive comment lines inside a function body
      C3  doc comment on an exported identifier not starting with its name

    With no file argument, checks every .go file changed in the working tree.

OPTIONS
    -h, --help          Show this help
    -v, --version       Show version
    --all               Check every .go file in the repository
    --max-len N         Override the C1 limit (default: $MAXLEN)

ARGUMENTS
    files...            .go files to check

    Exit 0 = clean, 1 = at least one violation, 2 = bad arguments or environment.

EXAMPLES
    bash $SCRIPT_NAME internal/app/product/command/create_product.go
    bash $SCRIPT_NAME                 # every .go file changed in the working tree
    bash $SCRIPT_NAME --all
EOF
}

MODE="changed"
FILES=()
while [[ $# -gt 0 ]]; do
    case "$1" in
        -h|--help)    usage; exit 0 ;;
        -v|--version) echo "$SCRIPT_NAME v$VERSION"; exit 0 ;;
        --all)        MODE="all"; shift ;;
        --max-len)    MAXLEN="${2:?}"; shift 2 ;;
        -*)           echo "error: unknown option: $1" >&2; usage >&2; exit 2 ;;
        *)            FILES+=("$1"); MODE="args"; shift ;;
    esac
done

command -v python3 >/dev/null 2>&1 || { echo "error: python3 required" >&2; exit 2; }

if [[ "$MODE" == "all" ]]; then
    while IFS= read -r f; do FILES+=("$f"); done < <(find . -name '*.go' -not -path './vendor/*' -not -path './.git/*' | sort)
elif [[ "$MODE" == "changed" ]]; then
    git rev-parse --is-inside-work-tree >/dev/null 2>&1 || { echo "error: not a git repo — pass files explicitly" >&2; exit 2; }
    while IFS= read -r f; do [[ -n "$f" && -f "$f" ]] && FILES+=("$f"); done < <(
        { git diff --name-only --diff-filter=ACMR -- '*.go'; git diff --cached --name-only --diff-filter=ACMR -- '*.go'; git ls-files --others --exclude-standard -- '*.go'; } | sort -u
    )
fi

[[ ${#FILES[@]} -eq 0 ]] && { echo "no .go file to check"; exit 0; }

python3 - "$MAXLEN" "${FILES[@]}" <<'PY'
import re, sys

MAXLEN = int(sys.argv[1])
paths = sys.argv[2:]

# C1 exemptions: layout is dictated by the tool that reads the line.
EXEMPT_LEN = re.compile(r'^\s*//(go:|\s*@|\t*@)')
EXPORTED = [
    re.compile(r'^func\s+([A-Z]\w*)\s*\('),
    re.compile(r'^func\s+\([^)]*\)\s+([A-Z]\w*)\s*\('),
    re.compile(r'^type\s+([A-Z]\w*)\b'),
    re.compile(r'^(?:var|const)\s+([A-Z]\w*)\b'),
]

def check(path):
    findings = []
    try:
        lines = open(path, encoding='utf-8').read().split('\n')
    except OSError as err:
        return [(0, 'ERR', f'cannot read: {err}')]

    if any('Code generated' in l and 'DO NOT EDIT' in l for l in lines[:5]):
        return findings
    is_test = path.endswith('_test.go')

    depth = 0
    run = run_start = 0

    def flush_run():
        if run > 2:
            findings.append((run_start, 'C2',
                             f'{run} consecutive comment lines inside a function body (max 2) — '
                             'move the explanation to the top of the file'))

    for i, line in enumerate(lines, 1):
        stripped = line.strip()
        iscomment = stripped.startswith('//')

        if iscomment and len(line) > MAXLEN and not EXEMPT_LEN.match(line):
            findings.append((i, 'C1', f'comment line is {len(line)} chars (max {MAXLEN})'))

        if iscomment and depth > 0:
            if run == 0:
                run_start = i
            run += 1
        else:
            flush_run()
            run = 0

        if not is_test:
            for rx in EXPORTED:
                m = rx.match(line)
                if not m:
                    continue
                if i < 2 or not lines[i - 2].strip().startswith('//'):
                    break
                j = i - 2
                while j >= 1 and lines[j - 1].strip().startswith('//'):
                    j -= 1
                first = lines[j].strip()[2:].strip()
                name = m.group(1)
                if first and not first.split()[0].rstrip(',:').startswith(name):
                    findings.append((j + 1, 'C3',
                                     f'doc comment should start with "{name}", starts with "{first.split()[0]}"'))
                break

        if not iscomment:
            depth = max(0, depth + line.count('{') - line.count('}'))

    flush_run()
    return findings

total = 0
for p in paths:
    for ln, code, msg in check(p):
        total += 1
        print(f"{p}:{ln}: {code} {msg}")

print()
if total == 0:
    print(f"comments OK ({len(paths)} file(s) checked)")
    sys.exit(0)
print(f"{total} comment violation(s) — rules: .agents/skills/go-style-core/SKILL.md (\"Comments\")")
sys.exit(1)
PY
