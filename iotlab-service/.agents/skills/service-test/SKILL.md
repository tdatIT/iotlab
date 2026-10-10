---
name: service-test
description: Mandatory test policy for iotlab-service backend services — Definition of Done, Gate A (unit line coverage >= 80%), Gate B (an e2e test per selected API). Use when building, testing, or reviewing any API.
allowed-tools: Bash(bash:*), Bash(go:*), Bash(docker:*), Bash(make:*)
---

# Mandatory Service Test Requirements

The **required** standard for every iotlab-service backend service. This skill defines **WHAT is
required and WHEN**; **HOW to write** the tests lives in
[writing-tests-go-projects](../writing-tests-go-projects/SKILL.md).

## Scripts

| Script | Enforces | Dependencies |
| --- | --- | --- |
| `scripts/coverage.sh --threshold 80` | **Gate A** — unit line coverage ≥ 80% | none |
| `scripts/e2e.sh` | **Gate B** — runs the e2e suite, then every selected API must have an e2e test | postgres/emqx5 |
| `scripts/list-apis.sh` | lists every API with its `SELECTED` / `E2E` status; `--gate` is Gate B on its own | none |
| `scripts/test-docker.sh` | runs Gate A (and Gate B with `--e2e`) inside `golang:<go.mod version>-alpine` | docker |

Makefile shortcuts: `make coverage` (Gate A), `make e2e` (Gate B), `make list-apis`.

> **CI does not enforce these gates.** The pipeline no longer runs a coverage job, so a green
> pipeline says nothing about coverage. Whoever opens the MR runs the gates by hand.

---

## 1. What is required

- **Unit test:** every API/handler, every command/query, every business branch **and** every error
  branch. Written per [writing-tests-go-projects](../writing-tests-go-projects/SKILL.md).
- **E2E test:** every API in the **selected e2e scope**. The default scope is **every business API**
  the service exposes; narrow it only with the user's explicit choice (§4).
- **Gate A — unit line coverage ≥ 80%** per service, after excluding generated/mock/`cmd/`/`test/e2e/`.
- **Gate B — e2e API coverage 100% of the selected list.** Not line coverage: it counts *how many
  selected APIs have an e2e test*.

The two gates are independent. Gate A runs without dependencies; Gate B needs the compose stack.

## 2. Definition of Done — what "finished" means

> **An API counts as COMPLETE only when its unit test passes, it is inside Gate A, and — if it is in
> the e2e scope — it has an e2e test.**
> Code that runs but has no test is **NOT finished**: it does not count as progress and does not go
> into a "Ready" MR.

Per API/handler:

- [ ] ≥ 1 unit test for the **happy path**.
- [ ] A test for **every error branch** (validation failure, not-found, duplicate, internal error…).
- [ ] An e2e test if the API is in the selected scope.
- [ ] `bash scripts/coverage.sh --threshold 80` PASSES.
- [ ] `bash scripts/e2e.sh` PASSES.

## 3. Unit tests are the feature's documentation

Write tests so that **reading the test explains the feature** without reading the code:

- Name the test after the behaviour: `Test<Subject>_<condition>_<expectation>`, e.g.
  `TestCreateProduct_DuplicateSKU_ReturnsAlreadyExists`.
- `t.Run("duplicate sku is rejected", ...)` — a subtest name is **one sentence of specification**.
- A table-driven test is a **specification table** of input → output, one row per case (see
  [test-table](../writing-tests-go-projects/resources/test-table.md)).
- One business/error branch = one named case. Never merge several behaviours into one vague test.

## 4. E2E tests

**Location and convention**

- The suite lives in `test/e2e/`, package `e2e`, one `*_test.go` file per domain.
- Every file starts with the build tag, so a normal `go test ./...` never touches the suite:

  ```go
  //go:build e2e

  package e2e
  ```

- `TestMain` boots the real service (`server.NewServer()` → `httptest.NewServer(app.API())`) against
  the compose dependencies. Helpers stay in `*_test.go` files so they never enter the Gate A
  denominator.
- Each test resets the state it touches (e.g. `resetProducts`) — never depend on execution order.
- Cover **critical flows** end to end (HTTP → handler → app → repo → DB); do not re-test every unit
  case here.

**Declaring what a test covers**

Each e2e test declares the API it covers with an annotation. This is what Gate B counts:

```go
// TestE2E_CreateProduct covers the create flow end to end.
//
// @e2e POST /api/v1/products
func TestE2E_CreateProduct(t *testing.T) { ... }
```

Without the annotation the API shows as `MISSING` even if a test exercises it.

**Choosing the scope — always show the list first**

Before writing e2e tests, run `bash scripts/list-apis.sh` and **show the table to the user so they
can pick the scope**:

```
#   METHOD  PATH                       HANDLER          PLATFORM  SELECTED  E2E
6   POST    /api/v1/products           CreateProduct    -         yes       yes
7   GET     /api/v1/products           ListProducts     -         yes       MISSING
```

- **The default is EVERY business API selected.** Only narrow it when the user says so.
- Platform routes (`/metrics`, `/liveness`, `/readiness`, `/swagger*`) are listed but never
  auto-selected.
- A narrowed scope is stored in `test/e2e/.e2e-apis`, one `METHOD PATH` per line, and is **committed**
  so the whole team shares one scope. No file = the default scope.

  ```
  # test/e2e/.e2e-apis — e2e scope; delete this file to go back to "all business APIs"
  POST /api/v1/products
  GET /api/v1/products/:id
  ```

## 5. Interleave the tests as you build (NOT big-bang)

Do **not** build a batch of APIs and come back for the tests later. Work the **per-API loop**:

```
For each API/handler:
  1. (if a spec/doc exists) Write the test FIRST from the spec -> red (TDD)
  2. Implement the minimum until it goes green
  3. Add the error-branch cases -> green
  4. Refactor (tests stay green)
  5. coverage.sh must pass before moving to the next API
```

Commit in thin slices where "the feature and its test travel together" — no commit or MR should
carry code whose tests are missing. Test debt never piles up, coverage never drops far, and every
MR proves itself.

## 6. TDD-first when a proper spec exists

When the feature already has a clear spec (OpenAPI/Postman, acceptance criteria, an API
description): **generate the unit tests from the spec first** so they act as the contract, then
develop until they are green. When the spec is unclear, write the tests alongside the code with the
loop in §5 — right after each branch, never at the end.

## 7. Run the gates on Docker

Run build + test in Docker so the toolchain matches `go.mod` exactly, instead of whatever the host has:

```sh
# Gate A (unit coverage) in docker
bash .agents/skills/service-test/scripts/test-docker.sh --threshold 80

# Gate A + Gate B with real postgres/emqx5
bash .agents/skills/service-test/scripts/test-docker.sh --e2e

# print the docker command without running it
bash .agents/skills/service-test/scripts/test-docker.sh --print
```

> Why Docker: coverage instrumentation needs the same `go` toolchain as `go.mod`. Running inside
> `golang:<version>-alpine` removes "version does not match" failures and keeps everyone's numbers
> comparable.

## Related

- [writing-tests-go-projects](../writing-tests-go-projects/SKILL.md) — how to write the unit tests
  (mocks, tables, black-box).
- [task-management](../task-management/SKILL.md) — the task's test checklist.
