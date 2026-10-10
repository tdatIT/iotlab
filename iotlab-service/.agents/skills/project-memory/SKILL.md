---
name: project-memory
description: Use to recall accumulated project experience before working, and to capture durable lessons AFTER mistakes/corrections or important decisions — with a strict quality gate so memory stays high-signal and never gets diluted. Trigger on "rút kinh nghiệm", "ghi nhớ", "remember this", "lesson learned", "vì sao lại sai", "đừng lặp lại", "memory", "curate memory", or whenever the learning-loop hook fires.
allowed-tools: Bash(bash:*)
---

# Project Memory & Learning Loop (local, chống loãng)

Bộ nhớ **kinh nghiệm theo project**, lưu local ở `.agents/memory/` (1 file = 1 ý). Mục tiêu:
agent **recall trước khi làm**, **rút kinh nghiệm sau khi sai**, và **giữ memory tinh gọn** — chỉ
điều thực sự đáng giá, không nhồi linh tinh làm loãng.

## Available Scripts

- **`scripts/memory.sh`** — `add` / `list` / `search` / `recall` / `index` / `curate`. Tự kiểm tra trùng + cập nhật
  index + cảnh báo loãng. `bash scripts/memory.sh --help`.

## Vòng lặp học (learning loop)

```
RECALL (hook nạp MEMORY.md đầu phiên · memory.sh search)
   → ACT (làm task)
   → MISTAKE/CORRECTION (hook memory-reflect bắt tín hiệu user sửa sai)
   → REFLECT (lỗi gốc là gì? lẽ ra nên biết gì?)
   → RECORD (memory.sh add — CHỈ khi qua GATE bên dưới)
   → CURATE (memory.sh curate — gộp/cắt định kỳ)
   → RECALL tốt hơn ở lần sau
```

## ✅ GATE — chỉ ghi khi qua **TẤT CẢ** 4 tiêu chí

1. **Durable** — đúng vượt ra ngoài task hiện tại (không phải state nhất thời).
2. **Non-obvious** — không suy ra được từ code/git/CLAUDE.md/docs đang có.
3. **Reusable** — sẽ còn gặp lại, ảnh hưởng quyết định tương lai.
4. **Actionable** — đổi được hành vi lần sau (có "how to apply").

Nghi ngờ ở bất kỳ tiêu chí nào → **KHÔNG ghi**. Thà ít mà chất.

## 🚫 KHÔNG bao giờ ghi (giữ memory không loãng)

- Chi tiết dùng-một-lần của task ("đổi tên biến X ở file Y").
- Thứ đã có trong code / git history / CLAUDE.md / GUIDELINE.
- Secret/credential — không bao giờ ghi vào memory, kể cả để "tham khảo".
- Cảm tính mơ hồ ("cẩn thận hơn"), không actionable.
- Trạng thái tạm (port đang chạy, TODO của hôm nay).

## Loại memory (type)

| type         | dùng cho                                         |
|--------------|--------------------------------------------------|
| `lesson`     | bài học rút sau khi sai/correction               |
| `decision`   | quyết định kiến trúc + lý do (ADR rút gọn)       |
| `gotcha`     | cạm bẫy/đặc tính dễ vấp của lib/hệ thống         |
| `convention` | quy ước nội bộ chưa nằm trong docs               |
| `reference`  | con trỏ tới tài nguyên ngoài (dashboard, ticket) |

## Ghi 1 memory

```sh
bash .agents/skills/project-memory/scripts/memory.sh add \
  --type lesson \
  --title "Readiness skipper dùng sai path" \
  --desc "metrics skipper phải khớp /liveness /readiness, không phải /live /ready" \
  --why "health endpoint bị tính vào metrics do path lệch" \
  --apply "khi đổi route health, sửa luôn skipper trong http.go" \
  --tags echo,metrics
```

- 1 file = **1 ý**. Có **why** + **how to apply** cho lesson/decision (phần khiến memory đáng giá).
- Liên kết memory liên quan trong nội dung bằng `[[ten-slug-khac]]`.
- `add` **không chặn cứng** khi nghi trùng (trừ trùng slug y hệt): nó **ghi + cảnh báo ứng viên** rồi để **agent tự gộp
  ngữ nghĩa** (xem mục dưới). `--strict` nếu muốn chặn thay vì cảnh báo.

## Phát hiện trùng & gộp — 2 tầng (cơ học vs ngữ nghĩa)

| Tầng                  | Làm gì                                                                                                                                   |
|-----------------------|------------------------------------------------------------------------------------------------------------------------------------------|
| **Script (cơ học)**   | Rẻ + tất định: gắn cờ **ứng viên** nghi trùng (chuỗi/tag), đo độ dài & số lượng, cảnh báo. **Không** hiểu nghĩa, **không** tự sửa prose. |
| **Agent (ngữ nghĩa)** | Đọc ứng viên → phán đoán **có thật sự cùng ý không** (kể cả khác chữ) → **gộp / tóm tắt / cắt** rồi đồng bộ index.                       |

> Vì sao tách: regex không hiểu *"Echo context is pointer"* ≡ *"handler nhận `*echo.Context`"*. Auto-merge bằng text
> rule sẽ bỏ sót (paraphrase) và gộp ẩu. Quyết định gộp = việc của agent.

### Quy trình agent gộp ngữ nghĩa (khi `add`/`curate` báo ứng viên, hoặc khi thấy memory loãng/dài)

1. Đọc nội dung các file ứng viên + file mới.
2. **Cùng một ý?** (cùng nguyên nhân/bài học, dù khác cách diễn đạt)
    - **Có** → gộp thành **1** memory: giữ phát biểu **tổng quát nhất**, **union** các `why`/`how to apply`, giữ
      `created` sớm nhất, hạ những điểm phụ; xoá (các) file thừa.
    - **Không** → giữ riêng, nhưng link nhau bằng `[[ ]]` cho dễ recall.
3. **Memory quá dài / lan man** → tóm tắt lại còn phần actionable; chi tiết dài bỏ bớt.
4. Đã thành code/test/CLAUDE.md (luật cứng) → **xoá** memory (đã thừa).
5. Chạy `memory.sh index` để đồng bộ `MEMORY.md`.

## Recall trước khi làm

```sh
bash .agents/skills/project-memory/scripts/memory.sh search redis
bash .agents/skills/project-memory/scripts/memory.sh recall echo     # in kèm nội dung
```

Hook `memory-load` (SessionStart) đã tự nạp index đầu phiên — đọc nó trước khi bắt tay vào việc liên quan.

## Curate (định kỳ — chống loãng)

```sh
bash .agents/skills/project-memory/scripts/memory.sh curate   # liệt kê ỨNG VIÊN: nghi trùng/cụm tag/quá dài/quá hạn/vượt ngưỡng
bash .agents/skills/project-memory/scripts/memory.sh index    # đồng bộ MEMORY.md sau khi agent sửa file
```

`curate` chỉ **liệt kê ứng viên** (cơ học). Việc **gộp/tóm tắt/cắt là agent làm** theo "Quy trình agent gộp ngữ nghĩa" ở
trên.

- Ngưỡng số lượng `MEMORY_CAP` (mặc định 60), độ dài `MEMORY_MAXLINES` (25), quá hạn `MEMORY_STALE_DAYS` (180) — chỉnh
  qua env.
- Vượt ngưỡng = dấu hiệu đang nhồi linh tinh → ưu tiên gộp/cắt.

## Tự động hoá (hook)

- `.claude/hooks/memory-load.sh` — SessionStart: nạp index để recall. Im lặng khi chưa có memory nào.
- `.claude/hooks/memory-reflect.sh` — UserPromptSubmit: **chỉ** khi prompt có tín hiệu sửa sai mới nhắc ghi
  lesson (qua GATE) — chạy mỗi lượt nên mặc định câm, không đốt token.

`.agents/memory/` chỉ xuất hiện sau lệnh `add` đầu tiên; trước đó hook không in gì.

> Memory là **local theo project** (`.agents/memory/`), commit cùng repo để cả team & agent dùng chung kinh nghiệm.
