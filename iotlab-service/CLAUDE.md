# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 0. Project Guideline

**iotlab-service** (`github.com/tdatIT/iotlab-service`): ESP32 devices publish over MQTT, workers store the data in
TimescaleDB, and a REST API serves it. See `README.md` for topics, payloads and endpoints.

**Read `docs/GUIDELINE.md` before implementing any feature**, together with:

- `.agents/skills/go-style-core/SKILL.md`
- `.agents/skills/go-clean-cqrs-architecture/SKILL.md`

They define the layer contracts (Handler/Worker → App → Repository), DTO/command/query/handler patterns, config and
error conventions. Do not implement a feature, add a layer, or create a new domain before reading the relevant section.

### 0.1 Where this repo differs from GUIDELINE.md and the skills

Those documents still describe the original template (GORM, Redis, tracing, the `product` sample). The code below wins:

| Topic | This repo |
| --- | --- |
| Persistence | **pgx v5**, no GORM. Repositories take `database.Querier` and write parameterized SQL (`$1…`); scan with `pgx.CollectRows` / `pgx.RowToAddrOfStructByName` into models with `db` tags. Never build SQL from input; quote dynamic identifiers with `pgx.Identifier{}.Sanitize()`. |
| Repository errors | Return `database.ErrNotFound` / `database.ErrDuplicate` via `database.Translate(err)`, never pgx errors. The app layer checks those sentinels and maps them to `svcerr.*`. |
| DB handle | `database.DB` → `Pool()` for repositories, `SQLDB()` only for golang-migrate and the readiness probe. |
| Schema | TimescaleDB migrations in `migrations/`. A continuous aggregate must be the only statement of its migration file. |
| Time | Store UTC (`time.UnixMilli(ms).UTC()`); range queries use DB time: `time >= now() - $2::interval` with a `time.Duration` arg. |
| JSON | `encoding/json/v2` (no sonic). Echo still renders HTTP responses with its own encoder. |
| Domain | One domain, `device`: `app/device/{command,query}`, `handler/device`, repositories `device`, `sensor`, `telemetry`, workers in `worker/event`. No `product` sample. |
| Not wired | Redis (`rdclient`, `cache`, `locker`) and tracing (`pkgs/tracing`, tracing middleware) stay as libraries only — no config keys, not in `server.go`. |
| Rate limit | Every API route gets its own limiter: `rateLimiter(routeLimit)` per route in `router/routes.go`, config `routeRateLimit`. |
| Config | `config/config.yaml` (defaults) and `config/config-dev.yaml`; select with `CONFIG_PATH`. Secrets stay empty in YAML and come from env vars. |
| Go | 1.27.1 — follow the Modern Go Guidelines (`new(expr)`, `errors.AsType`, `wg.Go`, `t.Context()`, …). |

### 0.2 Testing in this repo

- App / handler / worker tests: mockery mocks (`make mocks`). Variadic mock args are matched as **one slice**:
  `repo.EXPECT().Upsert(mock.Anything, mock.Anything, []string{"last_seen"})`.
- Repository tests: `pgxmock` (`pgxmock.NewPool()`), asserting SQL fragments and bound args. They do not execute SQL;
  behaviour against real TimescaleDB is covered by the e2e suite (`make e2e`).
- After changing a handler annotation run `make swagger`; after changing a repository interface run `make mocks`.

---

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:

- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:

- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:

- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Validate Once at the Right Layer

**Validate at the entry boundary. Trust layers below don't need to repeat it.**

- The layer that first receives external input owns validation. For HTTP handlers that means the DTO + Echo validator.
  For consumers it means the message deserialization step.
- **Application/service layers must NOT re-validate** what the caller already validated (nil checks, zero-value guards,
  required-field assertions). These checks add noise and imply distrust of the contract.
- Only add validation in a deeper layer when there is **genuine business logic** that layer uniquely owns — e.g. a
  cross-field invariant, an external-system constraint, or a rule that no caller can pre-check.
- Before adding any guard/check, ask: "has the caller already ensured this?" If yes, delete the check.

Rule of thumb per layer in this project:

| Layer | Validates |
| --- | --- |
| Handler | DTO binding + struct tags (required, min, format…) |
| App | Business invariants only — never nil/zero-value re-checks |
| Repository | Nothing — trusts the app layer |

## 5. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:

- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multistep tasks, state a brief plan:

1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and
clarifying questions come before implementation rather than after mistakes.
