#!/usr/bin/env bash
set -euo pipefail

# memory.sh — Quản lý "bộ nhớ kinh nghiệm" của project dưới .agents/memory/.
# Mỗi memory = 1 file = 1 ý (frontmatter + nội dung). MEMORY.md là index 1 dòng/memory.
# Mục tiêu: ghi có kỷ luật (chống loãng), dễ recall, tự dọn (curate).

VERSION="1.1.0"
SCRIPT_NAME="$(basename "$0")"
MEMORY_DIR="${MEMORY_DIR:-.agents/memory}"
INDEX="$MEMORY_DIR/MEMORY.md"
CAP="${MEMORY_CAP:-60}"          # số memory tối đa trước khi cảnh báo loãng
STALE_DAYS="${MEMORY_STALE_DAYS:-180}"
MAXLINES="${MEMORY_MAXLINES:-25}"  # 1 memory dài quá -> đề xuất tóm tắt

usage() {
    cat <<EOF
$SCRIPT_NAME v$VERSION — Quản lý memory kinh nghiệm local (.agents/memory/)

USAGE
    bash $SCRIPT_NAME <command> [options]

COMMANDS
    add        Thêm 1 memory (có kiểm tra trùng + cập nhật index)
    list       In index MEMORY.md
    search Q   Tìm memory theo từ khoá (name/description/body)
    index      Dựng lại MEMORY.md từ các file (đồng bộ)
    curate     Báo cáo sức khoẻ: trùng lặp, mồ côi, quá hạn, vượt ngưỡng loãng
    recall Q   Như search nhưng in kèm nội dung tóm tắt (để nạp vào ngữ cảnh)

ADD OPTIONS
    --type T       lesson|decision|gotcha|convention|reference   (bắt buộc)
    --title "..."  tiêu đề ngắn (bắt buộc)
    --desc "..."   1 dòng mô tả để recall (bắt buộc)
    --body "..."   nội dung chính (mặc định = desc)
    --why "..."    vì sao quan trọng (khuyến nghị cho lesson/decision)
    --apply "..."  áp dụng thế nào lần sau (khuyến nghị cho lesson)
    --tags a,b     nhãn, phân tách bằng dấu phẩy
    --confidence X high|medium|low (mặc định: high)
    --source "..." nguồn (PR/issue/file/session)
    --force        bỏ qua cảnh báo trùng

VÍ DỤ
    bash $SCRIPT_NAME add --type lesson \\
      --title "Echo v5 context is pointer" \\
      --desc "handler nhận *echo.Context, không phải echo.Context" \\
      --why "build fail nhiều lần do nhầm value receiver" \\
      --apply "luôn dùng c *echo.Context trong handler" --tags echo,go
    bash $SCRIPT_NAME curate
    bash $SCRIPT_NAME search redis

Exit 0 = ok, 1 = vi phạm/cảnh báo chặn, 2 = lỗi tham số.
EOF
}

slugify() {
    printf '%s' "$1" | tr '[:upper:]' '[:lower:]' \
        | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' | cut -c1-50
}

ensure_dir() {
    mkdir -p "$MEMORY_DIR"
    if [[ ! -f "$INDEX" ]]; then
        {
            echo "# MEMORY — Kinh nghiệm project (local)"
            echo ""
            echo "> 1 dòng/memory. Nội dung đầy đủ ở từng file. Quản lý bằng \`memory.sh\`."
            echo "> Chỉ ghi điều **bền vững · không hiển nhiên · tái dùng được**. Đừng làm loãng."
            echo ""
        } > "$INDEX"
    fi
}

today() { date +%F; }

frontmatter_field() {  # file field -> value
    sed -n "/^---$/,/^---$/p" "$1" | grep -E "^$2:" | head -1 | sed -E "s/^$2:[[:space:]]*//"
}

cmd_add() {
    local type="" title="" desc="" body="" why="" apply="" tags="" conf="high" source="" force=false strict=false
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --type) type="${2:?}"; shift 2 ;;
            --title) title="${2:?}"; shift 2 ;;
            --desc) desc="${2:?}"; shift 2 ;;
            --body) body="${2:?}"; shift 2 ;;
            --why) why="${2:?}"; shift 2 ;;
            --apply) apply="${2:?}"; shift 2 ;;
            --tags) tags="${2:?}"; shift 2 ;;
            --confidence) conf="${2:?}"; shift 2 ;;
            --source) source="${2:?}"; shift 2 ;;
            --force) force=true; shift ;;
            --strict) strict=true; shift ;;
            *) echo "error: unknown add option: $1" >&2; exit 2 ;;
        esac
    done
    [[ -z "$type" || -z "$title" || -z "$desc" ]] && { echo "error: cần --type, --title, --desc" >&2; exit 2; }
    case "$type" in lesson|decision|gotcha|convention|reference) ;; *)
        echo "error: --type phải là lesson|decision|gotcha|convention|reference" >&2; exit 2 ;; esac

    ensure_dir
    local slug; slug="$(slugify "$title")"
    local file="$MEMORY_DIR/$slug.md"
    [[ -z "$body" ]] && body="$desc"

    # (1) Trùng slug Y HỆT -> KHÔNG ghi đè (tránh mất dữ liệu). Đây là chặn cứng hợp lý.
    if [[ -f "$file" ]] && ! $force; then
        echo "⚠️  Đã có memory '$slug' — hãy CẬP NHẬT/GỘP file đó thay vì ghi đè:" >&2
        echo "    $file" >&2
        echo "    (--force nếu thực sự muốn ghi đè)" >&2
        exit 1
    fi

    # (2) Ứng viên nghi trùng (heuristic rẻ: 3 từ đầu slug). Đây CHỈ là tín hiệu cho agent,
    #     KHÔNG chặn — merge ngữ nghĩa là việc của agent (skill project-memory). --strict để chặn.
    local key3; key3="$(printf '%s' "$slug" | cut -d- -f1-3)"
    CANDIDATES=""
    if [[ -n "$key3" ]]; then
        CANDIDATES="$(grep -rl "$key3" "$MEMORY_DIR" 2>/dev/null | grep -v '/MEMORY.md' | grep -v "^$file$" || true)"
    fi
    if [[ -n "$CANDIDATES" ]] && $strict && ! $force; then
        echo "⚠️  [strict] Có ứng viên nghi trùng — không ghi. Gộp thủ công rồi thử lại:" >&2
        printf '%s\n' "$CANDIDATES" | sed 's/^/    /' >&2
        exit 1
    fi

    local tagline="[]"
    [[ -n "$tags" ]] && tagline="[$(printf '%s' "$tags" | sed 's/,/, /g')]"

    {
        echo "---"
        echo "name: $slug"
        echo "type: $type"
        echo "description: $desc"
        echo "tags: $tagline"
        echo "confidence: $conf"
        echo "created: $(today)"
        [[ -n "$source" ]] && echo "source: $source"
        echo "---"
        echo ""
        echo "$body"
        [[ -n "$why" ]]   && { echo ""; echo "**Why:** $why"; }
        [[ -n "$apply" ]] && { echo "**How to apply:** $apply"; }
    } > "$file"

    # Cập nhật index
    printf -- "- [%s](%s.md) — %s · _%s · %s_\n" "$title" "$slug" "$desc" "$type" "$(today)" >> "$INDEX"
    echo "✅ đã lưu: $file"
    if [[ -n "${CANDIDATES:-}" ]]; then
        echo ""
        echo "⚠️  Có memory cùng chủ đề (nghi trùng) — AGENT hãy review & GỘP NGỮ NGHĨA nếu thật sự trùng:"
        printf '%s\n' "$CANDIDATES" | sed 's/^/    /'
        echo "    → đọc các file trên + file vừa tạo; nếu cùng 1 ý: gộp thành 1 (giữ phát biểu tổng quát nhất,"
        echo "      union why/apply, link [[ ]]), xoá file thừa, rồi chạy 'memory.sh index'. Việc gộp là của agent, không phải script."
    fi
    cmd_curate_quiet
}

cmd_list() { ensure_dir; cat "$INDEX"; }

cmd_search() {
    local q="${1:?error: cần từ khoá}"
    ensure_dir
    local hits; hits="$(grep -rilF "$q" "$MEMORY_DIR" 2>/dev/null | grep -v "MEMORY.md" || true)"
    [[ -z "$hits" ]] && { echo "(không có memory khớp '$q')"; return 0; }
    while IFS= read -r f; do
        [[ -z "$f" ]] && continue
        printf "• %s — %s\n" "$(frontmatter_field "$f" name)" "$(frontmatter_field "$f" description)"
    done <<< "$hits"
}

cmd_recall() {
    local q="${1:?error: cần từ khoá}"
    ensure_dir
    local hits; hits="$(grep -rilF "$q" "$MEMORY_DIR" 2>/dev/null | grep -v "MEMORY.md" || true)"
    [[ -z "$hits" ]] && { echo "(không có memory khớp '$q')"; return 0; }
    while IFS= read -r f; do
        [[ -z "$f" ]] && continue
        echo "──── $(frontmatter_field "$f" name) [$(frontmatter_field "$f" type)] ────"
        sed '1,/^---$/d; /^---$/d' "$f" | sed '/^---$/d' | sed -n '1,12p'
        echo ""
    done <<< "$hits"
}

cmd_index() {
    ensure_dir
    local tmp="$INDEX.tmp"
    {
        echo "# MEMORY — Kinh nghiệm project (local)"
        echo ""
        echo "> 1 dòng/memory. Nội dung đầy đủ ở từng file. Quản lý bằng \`memory.sh\`."
        echo "> Chỉ ghi điều **bền vững · không hiển nhiên · tái dùng được**. Đừng làm loãng."
        echo ""
    } > "$tmp"
    local f
    for f in "$MEMORY_DIR"/*.md; do
        [[ -e "$f" ]] || continue
        [[ "$(basename "$f")" == "MEMORY.md" ]] && continue
        local name desc type created
        name="$(frontmatter_field "$f" name)"; desc="$(frontmatter_field "$f" description)"
        type="$(frontmatter_field "$f" type)"; created="$(frontmatter_field "$f" created)"
        printf -- "- [%s](%s.md) — %s · _%s · %s_\n" "${name}" "${name}" "${desc}" "${type}" "${created}" >> "$tmp"
    done
    mv "$tmp" "$INDEX"
    echo "✅ đã dựng lại index: $INDEX"
}

cmd_curate_quiet() {  # cảnh báo gọn sau add
    local n; n="$(find "$MEMORY_DIR" -maxdepth 1 -name '*.md' ! -name 'MEMORY.md' 2>/dev/null | grep -c . || true)"
    [[ "${n:-0}" -gt "$CAP" ]] && echo "⚠️  có $n memory (> ngưỡng $CAP) — chạy 'memory.sh curate' để gộp/cắt bớt, tránh loãng."
    return 0
}

cmd_curate() {
    ensure_dir
    local files; files="$(find "$MEMORY_DIR" -maxdepth 1 -name '*.md' ! -name 'MEMORY.md' 2>/dev/null | sort || true)"
    local n; n="$(printf '%s\n' "$files" | grep -c . || true)"
    echo "📋 Memory health — $MEMORY_DIR"
    echo "  Tổng: ${n:-0} memory (ngưỡng loãng: $CAP)"
    [[ "${n:-0}" -gt "$CAP" ]] && echo "  🚨 VƯỢT ngưỡng — gộp các memory cùng chủ đề, xoá cái lỗi thời."

    # Trùng lặp theo 3 từ đầu của slug
    echo ""; echo "  ▶ Nghi trùng (cùng 3 từ đầu tiêu đề):"
    local dup; dup="$(printf '%s\n' "$files" | while IFS= read -r f; do
        [[ -z "$f" ]] && continue
        b="$(basename "$f" .md)"; echo "$(printf '%s' "$b" | cut -d- -f1-3) $b"
    done | sort | awk '{c[$1]++; m[$1]=m[$1]" "$2} END{for(k in c) if(c[k]>1) print "    ["k"] ->"m[k]}')"
    [[ -n "$dup" ]] && printf '%s\n' "$dup" || echo "    (không có)"

    # Mồ côi: file thiếu trong index / index trỏ file không tồn tại
    echo ""; echo "  ▶ Lệch index:"
    local issue=false f b
    for f in $files; do
        [[ -z "$f" ]] && continue
        b="$(basename "$f")"
        grep -qF "($b)" "$INDEX" 2>/dev/null || { echo "    file thiếu trong index: $b"; issue=true; }
    done
    grep -oE '\(([a-z0-9-]+\.md)\)' "$INDEX" 2>/dev/null | tr -d '()' | while IFS= read -r ref; do
        [[ -f "$MEMORY_DIR/$ref" ]] || echo "    index trỏ file không tồn tại: $ref"
    done
    $issue && echo "    → chạy 'memory.sh index' để đồng bộ."

    # Quá hạn
    echo ""; echo "  ▶ Quá hạn (>$STALE_DAYS ngày — nên review/cập nhật/xoá):"
    local cutoff; cutoff="$(date -v-"${STALE_DAYS}"d +%F 2>/dev/null || date -d "-${STALE_DAYS} days" +%F 2>/dev/null || echo "")"
    if [[ -n "$cutoff" ]]; then
        local any=false
        for f in $files; do
            [[ -z "$f" ]] && continue
            c="$(frontmatter_field "$f" created)"
            [[ -n "$c" && "$c" < "$cutoff" ]] && { echo "    $(basename "$f") (created $c)"; any=true; }
        done
        $any || echo "    (không có)"
    else
        echo "    (không tính được mốc ngày trên hệ này)"
    fi
    # Memory quá dài -> đề xuất tóm tắt
    echo ""; echo "  ▶ Memory quá dài (> $MAXLINES dòng — nên tóm tắt gọn):"
    local longany=false lc
    for f in $files; do
        [[ -z "$f" ]] && continue
        lc="$(wc -l < "$f" | tr -d ' ')"
        [[ "${lc:-0}" -gt "$MAXLINES" ]] && { echo "    $(basename "$f") ($lc dòng)"; longany=true; }
    done
    $longany || echo "    (không có)"

    # Cụm tag trùng -> ứng viên cùng chủ đề (gợi ý gộp ngữ nghĩa)
    echo ""; echo "  ▶ Cụm theo tag (≥2 memory cùng tag — ứng viên gộp ngữ nghĩa):"
    local tagcluster
    tagcluster="$(for f in $files; do
        [[ -z "$f" ]] && continue
        b="$(basename "$f")"
        frontmatter_field "$f" tags | tr -d '[]' | tr ',' '\n' | while IFS= read -r one; do
            one="$(printf '%s' "$one" | tr -d ' ')"; [[ -n "$one" ]] && echo "$one|$b"
        done
    done | awk -F'|' '{c[$1]++; m[$1]=m[$1]" "$2} END{for(k in c) if(c[k]>1) print "    #"k" ->"m[k]}')"
    [[ -n "$tagcluster" ]] && printf '%s\n' "$tagcluster" || echo "    (không có)"

    echo ""
    echo "  ➡️  GỘP NGỮ NGHĨA là việc của AGENT (không phải script): đọc các ứng viên ở trên →"
    echo "     nếu cùng 1 ý thì gộp thành 1 (giữ phát biểu tổng quát nhất, union why/apply, link [[ ]]);"
    echo "     tóm tắt memory quá dài; xoá lesson đã thành code/test/CLAUDE.md; rồi 'memory.sh index'."
    echo "     Quy trình chi tiết: skill project-memory (mục Curate)."
}

CMD="${1:-}"; shift || true
case "$CMD" in
    -h|--help|help|"") usage ;;
    -v|--version) echo "$SCRIPT_NAME v$VERSION" ;;
    add)    cmd_add "$@" ;;
    list)   cmd_list ;;
    search) cmd_search "$@" ;;
    recall) cmd_recall "$@" ;;
    index)  cmd_index ;;
    curate) cmd_curate ;;
    *) echo "error: lệnh không hợp lệ: $CMD" >&2; usage >&2; exit 2 ;;
esac
