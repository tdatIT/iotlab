---
name: git-flow
description: Use when starting new work, naming a branch, or deciding how changes reach main in any iotlab-service repo — documents the team's trunk-based flow, branch naming, and merge rules. Trigger on "tạo nhánh", "branch name", "git flow", "luồng git", "bắt đầu task mới", "merge lên main".
---

# Luồng Git Của Team (Trunk-Based)

Repo iotlab-service dùng **trunk-based development**.
`main` luôn deploy được. Nhánh feature **ngắn hạn** (sống < 1–2 ngày), merge sớm, merge nhỏ.

## Sơ đồ

```
main (protected, luôn xanh)
  └─◄ feat/user-list-endpoint   ──MR (squash)──►  main
  └─◄ fix/httpclient-fd-leak    ──MR (squash)──►  main
```

- **KHÔNG** commit thẳng vào `main` (nhánh protected).
- Mỗi task = 1 nhánh feature từ `main` mới nhất.
- Hoàn thành → mở MR → review → **Squash merge** → xóa nhánh.
- Đồng bộ thường xuyên: `git fetch origin && git rebase origin/main` (tránh conflict lớn).

## Đặt tên nhánh

```
<type>/<scope>-<mô-tả-ngắn-kebab>
```

- `type` trùng bộ Conventional Commits: `feat` `fix` `chore` `refactor` `docs` `perf` `test` `ci`.
- Có thể gắn id task ở đầu: `feat/VM-123-user-list`.

| Tốt                        | Xấu          |
|----------------------------|--------------|
| `feat/user-list-endpoint`  | `new-branch` |
| `fix/VM-231-redis-timeout` | `quang-fix`  |
| `chore/bump-gorm`          | `test123`    |

## Bắt đầu một task

```sh
git checkout main && git pull --ff-only origin main
git checkout -b feat/user-list-endpoint
# ... code ...
# add file cụ thể (KHÔNG -A), scan secret, commit theo convention
# push nhánh rồi mở MR/PR vào main
```

## Quy tắc merge

- **Squash merge** để giữ lịch sử `main` tuyến tính, 1 commit/feature.
- Title commit squash = title MR (Conventional Commits).
- Bật **Remove source branch** + xóa nhánh sau merge.
- Không merge khi pipeline đỏ hoặc còn comment chưa resolve.

## Hotfix

Vẫn theo cùng luồng: `fix/...` từ `main`, MR ưu tiên review nhanh. Không tạo nhánh dài hạn riêng.

## Related

- [writing-commit-message](../writing-commit-message/SKILL.md)
