---
name: writing-commit-message
description: Use when staging files and writing a git commit for any iotlab-service repo — enforces Conventional Commits format and the team's git-add safety rules. Trigger on "commit", "viết commit", "commit message", "tạo commit".
allowed-tools: Bash(git status:*), Bash(git diff:*), Bash(git add:*), Bash(git commit:*), Bash(bash:*)
---

# Commit Message (Team Standard)

Shared commit standard for every iotlab-service repo. Goal: readable history, and never commit
someone else's files by accident.

## Required flow

```
1. git status + git diff --stat      → know exactly which files you changed
2. git add <explicit paths>          → never -A / . / --all
3. git diff --staged                 → review that the staged diff is only your change
4. git commit -m "<message>"         → verify with check-commit-msg.sh
```

Add only your own files. Never mix unrelated changes. A file touched by formatting only, unrelated to
the task → do not add.

## Format

```
<type>(<scope>): <subject>

[body — optional, explains WHY not WHAT, wrapped ~72 cols]

[footer — optional, e.g. Refs: #123, BREAKING CHANGE: ...]
```

- **type** (required): `feat` `fix` `docs` `style` `refactor` `perf` `test` `build` `ci` `chore` `revert`
- **scope** (recommended): affected area — domain or package name, e.g. `user`, `httpclient`, `config`
- **subject**: ≤ 72 chars, imperative mood (`add`, not `added`/`adds`), lowercase first letter, no trailing period
- **body**: required when the change is not self-evident
- **BREAKING CHANGE:** in footer when compatibility breaks

Good:

```
feat(user): add paginated list endpoint
fix(httpclient): close response body to stop fd leak
chore(deps): bump gorm to v1.31.2
```

Bad → fix:

| Bad             | Why                     | Fix                                            |
|-----------------|-------------------------|------------------------------------------------|
| `update code`   | no type, says nothing   | `refactor(config): simplify viper env binding` |
| `fixed bug`     | past tense + vague      | `fix(user): reject empty email on create`      |
| `feat: add user API and fix lint` | two intents in one | split into two commits            |

## Notes for Claude

- Derive `type`/`scope` from `git diff --staged`, never from memory.
- One commit = one intent. Two unrelated groups in the diff → propose splitting.
- Do not add `Co-Authored-By` / `Generated with` unless the repo requires it.
- Validate before committing: `bash scripts/check-commit-msg.sh "<message>"` (or `--file .git/COMMIT_EDITMSG`).
- This is enforced: the `commit-convention` hook runs that validator on every `git commit` and
  **denies** the command when the message breaks a rule (see
  [agent-setup](../agent-setup/SKILL.md)).

## Related

- [git-flow](../git-flow/SKILL.md) — branch naming & trunk-based flow.
