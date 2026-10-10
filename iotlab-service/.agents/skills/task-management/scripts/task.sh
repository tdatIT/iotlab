#!/usr/bin/env bash
set -euo pipefail

# task.sh — Quản lý task local cho agent dưới .agents/tasks/.
# Mỗi task = 1 file (TASK-NNNN.md) = define rõ TRƯỚC khi làm + gate khi complete.
# Vòng đời: todo → in_progress → completed (blocked/cancelled). Index ở TASKS.md.

VERSION="1.0.0"
SCRIPT_NAME="$(basename "$0")"
TASK_DIR="${TASK_DIR:-.agents/tasks}"
INDEX="$TASK_DIR/TASKS.md"

usage() {
    cat <<EOF
$SCRIPT_NAME v$VERSION — Quản lý task local cho agent (.agents/tasks/)

USAGE
    bash $SCRIPT_NAME <command> [options]

COMMANDS
    new       Tạo task mới (define rõ trước khi làm)
    list      In index TASKS.md
    show ID   In nội dung 1 task (vd: show TASK-0001)
    start ID  Chuyển todo → in_progress (kèm --branch)
    complete  Đóng task (GATE: checklist xong + summary + commit hash)
    index     Dựng lại TASKS.md từ các file

NEW OPTIONS (bắt buộc define rõ)
    --title "..."     tiêu đề ngắn            (bắt buộc)
    --why "..."       NGUYÊN NHÂN tại sao làm (bắt buộc)
    --goal "..."      MỤC ĐÍCH cần đạt        (bắt buộc)
    --output "..."    OUTPUT kỳ vọng          (bắt buộc)
    --confirm "a;b"   checklist hoàn thành (ngăn cách ';')
    --test "x;y"      checklist test (ngăn cách ';')
    --tags a,b

COMPLETE OPTIONS
    --id TASK-NNNN    task cần đóng            (bắt buộc)
    --summary "..."   tóm tắt đã làm           (bắt buộc)
    --commit HASH     commit hash              (bắt buộc)
    --branch NAME     nhánh
    --force           bỏ qua cảnh báo checklist còn dở

VÍ DỤ
    bash $SCRIPT_NAME new --title "Order CRUD P1" \\
      --why "service chưa quản lý đơn hàng" --goal "CRUD đơn hàng + publish event" \\
      --output "REST /api/v1/orders + test" \\
      --confirm "model Order;repo;handler;routes" --test "unit app;repo sqlite;coverage 90"
    bash $SCRIPT_NAME start TASK-0001 --branch feat/order-crud
    bash $SCRIPT_NAME complete --id TASK-0001 --summary "..." --commit a1b2c3d --branch feat/order-crud

Exit 0 = ok, 1 = gate chặn/cảnh báo, 2 = lỗi tham số.
EOF
}

ensure_dir() {
    mkdir -p "$TASK_DIR"
    [[ -f "$INDEX" ]] || {
        printf '# TASKS — Task log của project (local)\n\n> 1 dòng/task. Chi tiết ở từng file TASK-NNNN.md. Quản lý bằng `task.sh`.\n> Define rõ TRƯỚC khi làm; chỉ `completed` khi checklist xong + có summary + commit hash.\n\n' > "$INDEX"
    }
}

today() { date +%F; }

next_id() {
    local max=0 n f
    for f in "$TASK_DIR"/TASK-*.md; do
        [[ -e "$f" ]] || continue
        n="$(basename "$f" .md | sed 's/TASK-//' | sed 's/^0*//')"
        [[ -z "$n" ]] && n=0
        [[ "$n" -gt "$max" ]] && max="$n"
    done
    printf 'TASK-%04d' "$((max + 1))"
}

fm() { sed -n "/^---$/,/^---$/p" "$1" | grep -E "^$2:" | head -1 | sed -E "s/^$2:[[:space:]]*//"; }

# set 1 field trong frontmatter (block --- đầu tiên) — portable, không dùng sed -i
set_field() {
    local f="$1" k="$2" v="$3" tmp="$1.tmp"
    awk -v k="$k" -v v="$v" '
        /^---$/{d++}
        { if (d==1 && $0 ~ "^"k":" && !done){ print k": "v; done=1 } else print }
    ' "$f" > "$tmp" && mv "$tmp" "$f"
}

checklist_lines() {  # "a;b;c" -> "- [ ] a\n- [ ] b"
    local s="$1"
    [[ -z "$s" ]] && { echo "- [ ] (điền tiêu chí)"; return; }
    printf '%s' "$s" | tr ';' '\n' | while IFS= read -r it || [[ -n "$it" ]]; do
        it="$(printf '%s' "$it" | sed -E 's/^[[:space:]]+//; s/[[:space:]]+$//')"
        [[ -n "$it" ]] && echo "- [ ] $it"
    done
}

cmd_new() {
    local title="" why="" goal="" output="" confirm="" test="" tags=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --title) title="${2:?}"; shift 2 ;;
            --why) why="${2:?}"; shift 2 ;;
            --goal) goal="${2:?}"; shift 2 ;;
            --output) output="${2:?}"; shift 2 ;;
            --confirm) confirm="${2:?}"; shift 2 ;;
            --test) test="${2:?}"; shift 2 ;;
            --tags) tags="${2:?}"; shift 2 ;;
            *) echo "error: unknown new option: $1" >&2; exit 2 ;;
        esac
    done
    [[ -z "$title" || -z "$why" || -z "$goal" || -z "$output" ]] && {
        echo "error: PHẢI define rõ: --title, --why, --goal, --output" >&2; exit 2; }
    ensure_dir
    local id; id="$(next_id)"
    local file="$TASK_DIR/$id.md"
    local tagline="[]"; [[ -n "$tags" ]] && tagline="[$(printf '%s' "$tags" | sed 's/,/, /g')]"

    {
        echo "---"
        echo "id: $id"
        echo "title: $title"
        echo "status: todo"
        echo "tags: $tagline"
        echo "created: $(today)"
        echo "started:"
        echo "completed:"
        echo "branch:"
        echo "commit:"
        echo "---"
        echo ""
        echo "## Nguyên nhân / Why"
        echo "$why"
        echo ""
        echo "## Mục đích / Goal"
        echo "$goal"
        echo ""
        echo "## Output kỳ vọng"
        echo "$output"
        echo ""
        echo "## Checklist hoàn thành (confirm)"
        checklist_lines "$confirm"
        echo ""
        echo "## Checklist test"
        checklist_lines "$test"
        echo ""
        echo "## Tóm tắt đã làm"
        echo "_(điền khi complete)_"
    } > "$file"

    printf -- "- [%s] %s — _todo · %s_\n" "$id" "$title" "$(today)" >> "$INDEX"
    echo "✅ tạo task: $file"
    echo "→ sửa file để tinh chỉnh checklist trước khi 'start'."
}

cmd_list() { ensure_dir; cat "$INDEX"; }

cmd_show() {
    local id="${1:?error: cần ID}"
    local f="$TASK_DIR/$id.md"
    [[ -f "$f" ]] || { echo "error: không thấy $id" >&2; exit 2; }
    cat "$f"
}

cmd_start() {
    local id="${1:?error: cần ID}"; shift || true
    local branch=""
    while [[ $# -gt 0 ]]; do case "$1" in --branch) branch="${2:?}"; shift 2 ;; *) shift ;; esac; done
    local f="$TASK_DIR/$id.md"
    [[ -f "$f" ]] || { echo "error: không thấy $id" >&2; exit 2; }
    set_field "$f" status "in_progress"
    set_field "$f" started "$(today)"
    [[ -n "$branch" ]] && set_field "$f" branch "$branch"
    cmd_index >/dev/null
    echo "▶ $id → in_progress${branch:+ (branch $branch)}"
}

cmd_complete() {
    local id="" summary="" commit="" branch="" force=false
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --id) id="${2:?}"; shift 2 ;;
            --summary) summary="${2:?}"; shift 2 ;;
            --commit) commit="${2:?}"; shift 2 ;;
            --branch) branch="${2:?}"; shift 2 ;;
            --force) force=true; shift ;;
            *) echo "error: unknown complete option: $1" >&2; exit 2 ;;
        esac
    done
    [[ -z "$id" || -z "$summary" || -z "$commit" ]] && {
        echo "error: complete PHẢI có --id, --summary, --commit (hash)" >&2; exit 2; }
    local f="$TASK_DIR/$id.md"
    [[ -f "$f" ]] || { echo "error: không thấy $id" >&2; exit 2; }

    # GATE: còn checklist chưa tick → chặn (trừ --force)
    local pending; pending="$(grep -nE '^- \[ \] ' "$f" || true)"
    if [[ -n "$pending" ]] && ! $force; then
        echo "🚫 Chưa thể complete — còn checklist chưa hoàn thành:" >&2
        printf '%s\n' "$pending" | sed 's/^/    /' >&2
        echo "    → tick hết ([x]) rồi complete lại, hoặc --force nếu thật sự bỏ qua." >&2
        exit 1
    fi

    set_field "$f" status "completed"
    set_field "$f" completed "$(today)"
    set_field "$f" commit "$commit"
    [[ -n "$branch" ]] && set_field "$f" branch "$branch"
    {
        echo ""
        echo "## Tóm tắt đã làm (completed $(today))"
        echo "$summary"
        echo ""
        echo "- commit: \`$commit\`${branch:+ · branch: \`$branch\`}"
    } >> "$f"
    cmd_index >/dev/null
    echo "✅ $id → completed · commit $commit"
}

cmd_index() {
    ensure_dir
    local tmp="$INDEX.tmp" f
    printf '# TASKS — Task log của project (local)\n\n> 1 dòng/task. Chi tiết ở từng file TASK-NNNN.md. Quản lý bằng `task.sh`.\n> Define rõ TRƯỚC khi làm; chỉ `completed` khi checklist xong + có summary + commit hash.\n\n' > "$tmp"
    for f in "$TASK_DIR"/TASK-*.md; do
        [[ -e "$f" ]] || continue
        local id title status created commit
        id="$(fm "$f" id)"; title="$(fm "$f" title)"; status="$(fm "$f" status)"
        created="$(fm "$f" created)"; commit="$(fm "$f" commit)"
        printf -- "- [%s] %s — _%s · %s_%s\n" "$id" "$title" "$status" "$created" "${commit:+ · \`$commit\`}" >> "$tmp"
    done
    mv "$tmp" "$INDEX"
    echo "✅ đã dựng lại index: $INDEX"
}

CMD="${1:-}"; shift || true
case "$CMD" in
    -h|--help|help|"") usage ;;
    -v|--version) echo "$SCRIPT_NAME v$VERSION" ;;
    new) cmd_new "$@" ;;
    list) cmd_list ;;
    show) cmd_show "$@" ;;
    start) cmd_start "$@" ;;
    complete) cmd_complete "$@" ;;
    index) cmd_index ;;
    *) echo "error: lệnh không hợp lệ: $CMD" >&2; usage >&2; exit 2 ;;
esac
