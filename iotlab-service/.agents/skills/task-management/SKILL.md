---
name: task-management
description: Use BEFORE starting any non-trivial piece of work and to track it to completion — create a local task (why/goal/expected output/confirm checklist/test checklist) first, work it, then complete it with a summary + git commit hash. Trigger on "tạo task", "bắt đầu làm", "task mới", "định nghĩa task", "đánh dấu xong", "complete task", "task done", or whenever beginning a feature/fix.
allowed-tools: Bash(bash:*), Bash(git rev-parse:*), Bash(git log:*)
---

# Quản Lý Task Local (define trước · complete có gate)

Mọi việc không-tầm-thường **PHẢI có task** ở `.agents/tasks/` (1 file = 1 task). Mục tiêu: định nghĩa rõ
**trước khi làm**, và chỉ gọi là **xong** khi checklist đạt + có summary + đã commit (kèm commit hash).

## Available Scripts

- **`scripts/task.sh`** — `new` / `list` / `show` / `start` / `complete` / `index`. `bash scripts/task.sh --help`.

## Vòng đời

```
todo ──(start)──► in_progress ──(complete: gate)──► completed
                       └────────────► blocked / cancelled (khi cần)
```

## BƯỚC 1 — Tạo task TRƯỚC KHI làm (define rõ)

**Việc đầu tiên của mọi task** là tạo nó. Bắt buộc define đủ 4 thứ (script chặn nếu thiếu):

```sh
bash .agents/skills/task-management/scripts/task.sh new \
  --title "..." \
  --why    "NGUYÊN NHÂN — vì sao cần làm (vấn đề/bối cảnh)" \
  --goal   "MỤC ĐÍCH — cần đạt được gì" \
  --output "OUTPUT kỳ vọng — sản phẩm cụ thể bàn giao" \
  --confirm "tiêu chí 1;tiêu chí 2;..." \
  --test    "test 1;test 2;coverage..."
```

- Sinh `TASK-NNNN.md` (status `todo`) với khung: Why · Goal · Output · **Checklist hoàn thành** · **Checklist test** ·
  Tóm tắt.
- Mở file tinh chỉnh checklist cho sát thực tế trước khi `start`.

## BƯỚC 2 — Bắt đầu

```sh
bash .agents/skills/task-management/scripts/task.sh start TASK-0001 --branch feat/...
```

→ `in_progress` + ghi nhánh. (Tạo nhánh feat theo [git-flow](../git-flow/SKILL.md).)

## BƯỚC 3 — Làm & tick checklist

Làm theo TDD/quy ước stack. Khi đạt từng tiêu chí, **sửa file task** đổi `- [ ]` → `- [x]`
(cả checklist hoàn thành lẫn checklist test). Đây là bằng chứng để qua gate.

## BƯỚC 4 — Complete (có GATE)

Chỉ được gọi **completed** khi:

1. ✅ **Mọi checklist đã `[x]`** (script chặn nếu còn `[ ]`).
2. ✅ Test pass (theo [service/web/android-test-requirements]).
3. ✅ Đã **commit code** (theo [writing-commit-message](../writing-commit-message/SKILL.md)) → lấy hash.
4. ✅ Có **tóm tắt đã làm**.

```sh
HASH=$(git rev-parse --short HEAD)
bash .agents/skills/task-management/scripts/task.sh complete \
  --id TASK-0001 --summary "tóm tắt những gì đã làm" --commit "$HASH" --branch feat/...
```

→ status `completed`, ghi `commit`/`completed` vào frontmatter + append tóm tắt + commit hash vào file.

> Thứ tự đúng: **tick checklist → commit code → complete task (kèm hash)**. Commit trước, complete sau, để có hash thật.
>
> Hook `task-commit-hash` (PostToolUse Bash) tự in hash + lệnh `complete` ngay sau mỗi `git commit`, kèm số
> checklist còn `[ ]`. Nó chỉ nhắc, không tự đóng task.

## Định nghĩa "COMPLETED TASK" (DoD)

Một task chỉ xong khi: **checklist (confirm + test) đều `[x]` · test pass · code đã commit · commit hash ghi vào task ·
có tóm tắt**. Thiếu 1 → chưa xong.

## Granularity — khi nào 1 task, khi nào tách/gom (tránh quá detail)

**Nguyên tắc cốt lõi: 1 task = 1 đơn vị ship được = 1 MR. Chi tiết sống ở CHECKLIST, KHÔNG đẻ task con.**

```
Trivial (typo, đổi tên biến, 1-line, trả lời câu hỏi)        → KHÔNG tạo task, cứ làm
1 đơn vị ship/review được như 1 MR (1 endpoint/bug/feature nhỏ) → 1 TASK
   └─ các bước trong đó (model/repo/handler/test)             → CHECKLIST ITEMS (không phải task riêng)
Quá lớn (>~400 dòng / nhiều concern không liên quan / nhiều ngày) → TÁCH nhiều task, link [[...]]
Epic / phase (P1/P2/P3)                                        → nhóm nhiều task (ghi ở REQUIREMENTS / parent task)
```

- **GOM** khi các việc phụ thuộc nhau, ship cùng nhau, tách ra thì vô nghĩa/tạo trạng thái dở → 1 task (vd "
  model+repo+handler+test cho CRUD" = 1 task, không phải 4).
- **TÁCH** khi mỗi mảnh tự ship/test/review độc lập như 1 MR riêng.
- **Tie-break:** định tạo task con cho từng bước? → đó là dấu hiệu chúng nên là **checklist items** của 1 task.

## Branch — khi nào tạo nhánh khi `start`

Quyết bằng 1 câu: **"Task này có sinh commit vào repo dùng chung không?"**
| Tình huống | `start` |
|---|---|
| Task sinh code/doc commit | `start TASK-N --branch feat/...` — **luôn nhánh riêng** (main protected); type theo task
`feat/fix/chore/docs/refactor` |
| Sub-work của 1 MR đang mở | `start TASK-N --branch <nhánh-đang-mở>` — **dùng lại**, không tạo mới |
| Spike/điều tra/không commit | `start TASK-N` (không `--branch`); output = finding/doc/memory; `complete` dùng
`--force` (không có commit code) |

→ Không bao giờ commit thẳng `main`. Có commit ⇒ có nhánh.

## Quy ước

- Task **commit cùng repo** (`.agents/tasks/`) — log công khai cho team & agent.
- Recall: `task.sh list` để xem việc đang dở; `show TASK-NNNN` xem chi tiết.

## Related

- [git-flow](../git-flow/SKILL.md) · [writing-commit-message](../writing-commit-message/SKILL.md) · [service-test](../service-test/SKILL.md) · [project-memory](../project-memory/SKILL.md)
