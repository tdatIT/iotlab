---
name: agent-setup
description: How this repo drives Claude Code (CLAUDE.md, AGENTS.md, .agents/skills) plus a verify script for a fresh clone. Use when onboarding a machine, adding a skill, or when an agent ignores the repo conventions.
---

# Agent Setup

Everything that makes Claude Code follow this repo's conventions, and how to check it works on a
new machine.

```sh
bash .agents/skills/agent-setup/scripts/verify-setup.sh
```

Full walkthrough: [references/SETUP.md](references/SETUP.md).

## What the setup consists of

| Piece | Purpose | Loaded by |
| --- | --- | --- |
| `CLAUDE.md` | Behavioural rules (think first, simplicity, surgical changes, validate once, verify) | Claude Code, automatically |
| `AGENTS.md` | Same rules for other agents (Cursor, Codex, …) | those agents, automatically |
| `.agents/skills/*/SKILL.md` | Per-topic conventions (test policy, CQRS layout, git flow, commits…) | Claude Code, via the `.claude/skills` symlink |
| `.agents/skills/*/scripts/*.sh` | Executable gates and helpers | you or the agent, explicitly |
| `.claude/settings.json` + `.claude/hooks/` | Automatic formatting and guard rails | Claude Code, automatically |

## `.claude/` — what is automatic

`.claude/skills` is a symlink to `../.agents/skills`, so all 15 skills show up in Claude Code's
skill list while `.agents/` stays the single source of truth for other agents.

| Hook | Event | Does |
| --- | --- | --- |
| `memory-load.sh` | SessionStart | Injects the project memory index and any open task. Silent when there is nothing to recall. |
| `memory-reflect.sh` | UserPromptSubmit | Only when the prompt reads as a correction: reminds the agent to record the lesson. Silent otherwise. |
| `gofmt-on-write.sh` | PostToolUse `Edit\|Write` | gofmt any `.go` file just written. Advisory, never blocks. |
| `comment-rules.sh` | PostToolUse `Edit\|Write` | Checks the comments of a written `.go` file against [go-style-core](../go-style-core/SKILL.md); reports violations back to the agent. |
| `guard-git-add.sh` | PreToolUse `Bash` | **Blocks** `git add -A` / `git add .` / `git add --all` (exit 2). Stage explicit paths. |
| `commit-convention.sh` | PreToolUse `Bash` | **Blocks** a `git commit` whose message breaks Conventional Commits, via [writing-commit-message](../writing-commit-message/SKILL.md)'s validator. |
| `swagger-current.sh` | PreToolUse `Bash` | **Blocks** a `git commit` whose route changes are not reflected in `docs/swagger/` (see [swagger-api](../swagger-api/SKILL.md)). |
| `task-commit-hash.sh` | PostToolUse `Bash` | After a `git commit`, hands over the fresh hash and the command that closes the open task. Advisory. |
| `remind-test-gate.sh` | Stop | Warns when production Go changed with no test change. Advisory. |

Hooks fire **only when Claude Code is opened at this repo root** — the directory holding
`.claude/`. Opening a parent folder loads neither the hooks nor `CLAUDE.md`.

Two skills are additionally force-loaded through [CLAUDE.md](../../../CLAUDE.md) section 0
(`go-style-core`, `go-clean-cqrs-architecture`); add a skill there only when it must apply to
every single session.

## What verify-setup.sh checks

1. **Repo root** — anchored on `go.mod` + `.git`, not on agent config.
2. **Instruction files** — `CLAUDE.md`, `AGENTS.md`, `docs/GUIDELINE.md` present; reports which
   skills CLAUDE.md actually wires in.
3. **Skills** — every `.agents/skills/*/` has a `SKILL.md` with `name:` matching its directory, and
   no dead relative links.
4. **Tools** — required `bash`/`git`/`go`; optional `gofmt golangci-lint docker jq python3`.
5. **Skill scripts** — every `scripts/*.sh` answers `--help` and passes `bash -n`.
6. **Gates** — `make coverage` and `make e2e` resolve to real scripts (see
   [service-test](../service-test/SKILL.md)).
7. **Hooks** — `.claude/settings.json` present and every hook script parses.

Reports `PASS/WARN/FAIL`, exits 1 on any FAIL.

## Related

- [references/SETUP.md](references/SETUP.md) — step-by-step setup for a fresh clone.
- [service-test](../service-test/SKILL.md) — the test gates verify-setup resolves.
- [task-management](../task-management/SKILL.md) — start work through a task.
