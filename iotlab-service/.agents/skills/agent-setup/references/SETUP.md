# Claude Code Setup — This Repo, As It Is Today

Steps for a fresh clone.

## 1. Open Claude Code at the repo root

```sh
cd iotlab-service
claude
```

The root is where `go.mod`, `CLAUDE.md` and `AGENTS.md` live. Opening a parent directory (a
multi-repo folder) means `CLAUDE.md` is not picked up and the agent works without the repo's rules.

## 2. Know what loads automatically, and what does not

**Loads by itself**

- `CLAUDE.md` — behavioural rules for Claude Code.
- `AGENTS.md` — the same rules for other agents.
- `.claude/settings.json` — the three hooks (see step 3).
- `.agents/skills/*/SKILL.md` — surfaced through the `.claude/skills` symlink, so every skill is
  in Claude Code's skill list. `CLAUDE.md` section 0 additionally force-loads `go-style-core` and
  `go-clean-cqrs-architecture` in every session.

Available skills:

| Skill | Read it when |
| --- | --- |
| `go-style-core` | writing any Go code (wired in CLAUDE.md) |
| `go-clean-cqrs-architecture` | adding a domain, layer, handler (wired in CLAUDE.md) |
| `service-test` | any API — test policy, DoD, the two gates |
| `writing-tests-go-projects` | writing the unit tests themselves (mocks, tables) |
| `swagger-api` | annotating endpoints, regenerating the OpenAPI spec |
| `go-context` · `go-linting` · `naming-convention` · `handle-logging` | the matching concern |
| `git-flow` · `writing-commit-message` | branching, committing |
| `task-management` · `project-memory` | tracking work, capturing lessons |
| `agent-setup` | this document |

To make a skill apply to *every* session, add it to the CLAUDE.md section 0 list. Otherwise the
agent picks it up from the skill list when the task matches.

## 3. Know what the hooks do

| Hook | Event | Behaviour |
| --- | --- | --- |
| `memory-load.sh` | at session start | injects the memory index and open tasks so the agent recalls before working. |
| `memory-reflect.sh` | on every prompt | reminds the agent to record a lesson **only** when the prompt reads as a correction. |
| `gofmt-on-write.sh` | after every `Edit`/`Write` | runs `gofmt -w` on a `.go` file. Advisory. |
| `comment-rules.sh` | after every `Edit`/`Write` | checks Go comments against go-style-core (length, run length in a body, doc-comment prefix). |
| `guard-git-add.sh` | before every `Bash` | **blocks** `git add -A` / `git add .` / `git add --all`. Stage explicit paths. |
| `commit-convention.sh` | before every `Bash` | **blocks** a `git commit` with a non-conventional message (runs the writing-commit-message validator). |
| `swagger-current.sh` | before every `Bash` | **blocks** a `git commit` that changes routes/handlers/DTOs without a regenerated OpenAPI spec. |
| `task-commit-hash.sh` | after every `Bash` | after a `git commit`, prints the hash plus the `task.sh complete` command. |
| `remind-test-gate.sh` | when the agent stops | warns if production Go changed with no test change. Advisory. |

They fire only when Claude Code is opened **at the repo root** (the directory holding `.claude/`).
`.claude/settings.local.json` is git-ignored — put per-developer overrides there, never in
`settings.json`.

## 4. Install the toolchain

Required:

```sh
go version          # must match the go.mod line (currently go 1.26.2)
git --version
```

Recommended — each unlocks one workflow:

| Tool | Needed for |
| --- | --- |
| `gofmt` (ships with Go) | `make fmt` |
| `golangci-lint` | `make lint` |
| `docker` + `docker compose` | `make e2e`, `test-docker.sh` |
| `jq` or `python3` | JSON output of the skill scripts |
| `swag` | `make swagger` (`go install github.com/swaggo/swag/cmd/swag@v1.16.6`) |
| `migrate` | `make migrate-create` |

## 5. Bootstrap the project

```sh
go mod download
make mocks            # generates the mocks; do this before running tests on a clean checkout
make test             # everything must pass
make coverage         # Gate A: unit coverage >= 80%
```

`make mocks` matters on a fresh clone: the `mock/` packages do not exist yet, so any test importing
them breaks the package graph. Generate them first.

## 6. Start the dependencies (only for e2e / running the service)

```sh
docker compose up -d postgres redis emqx5
make e2e              # Gate B: runs test/e2e, requires an e2e test per selected API
make run              # or run the service itself
```

Unit tests need none of this — they are fully mocked.

## 7. Verify

```sh
bash .agents/skills/agent-setup/scripts/verify-setup.sh
```

Exit 0 means nothing is broken. Warnings are optional tools you can live without.

## Adding to the setup

- **New skill:** create `.agents/skills/<name>/SKILL.md` with frontmatter `name:` equal to the
  directory name and a one-line `description:`. Reference it from a related skill so it is
  discoverable. Add it to CLAUDE.md section 0 only if it must always apply.
- **New script:** put it in `.agents/skills/<skill>/scripts/`, support `--help`, exit 0 on success
  and non-zero on failure. verify-setup checks both.
- **New hook:** add the script to `.claude/hooks/`, wire it in `.claude/settings.json` under its
  event, and test it the way Claude Code calls it — the payload arrives as JSON on stdin:

  ```sh
  echo '{"tool_name":"Bash","tool_input":{"command":"git add -A"}}' | bash .claude/hooks/guard-git-add.sh
  ```

  Exit 2 in a `PreToolUse` hook denies the tool call; any other hook should exit 0 and stay
  advisory.
