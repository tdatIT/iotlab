---
name: swagger-api
description: >-
  Keep the OpenAPI spec in step with the routes. Use when adding or changing any endpoint, DTO or
  path param — annotate the handler, run `make swagger`, stage `docs/swagger/` in the same commit.
  A commit that changes the contract without a regenerated spec is blocked.
---

# Swagger API Workflow

Use this skill when updating the Swagger/OpenAPI documentation of this service. The template uses
Echo v5, routes live in `internal/router/routes.go`, handlers live in `internal/handler/<domain>`.

**Swagger is already wired.** `docs/swagger/` is generated and committed, `internal/router/routes.go`
blank-imports it and serves `/swagger` + `/swagger/*`, and `make swagger` regenerates it. Section 1
below is history — you only need it when standing up a *new* service. For day-to-day work start at
section 2.

The spec is enforced: `scripts/check-swagger.sh` compares the routes against `docs/swagger/swagger.json`,
and the `swagger-current` hook **blocks a `git commit`** that changes the contract without a
regenerated spec.

```sh
make swagger                                                  # regenerate
bash .agents/skills/swagger-api/scripts/check-swagger.sh      # is the spec current?
```

## Expected Outcome

Done means all five hold — the first three are checked by a command, so run it rather than assume:

| Outcome | How it is verified |
| --- | --- |
| Every business route has a matching path+method in `docs/swagger/swagger.json` | `make check-swagger` exits 0 |
| The regenerated spec ships with the change that caused it | `swagger-current` hook, on `git commit` |
| The annotated code still builds | `go build ./...` |
| Schemas name real DTOs — request, response and every documented error | read the generated diff |
| `/swagger` serves the updated UI locally | `make run`, open <http://localhost:3000/swagger> |

What "complete annotations" means here: `@Summary`, `@Tags`, `@Accept`/`@Produce` where a body is
involved, `@Param` for every path/query/body input, `@Success`, **one `@Failure` per status the
handler can actually return**, and `@Router`. A route documented with only the happy path is not
done — the error envelope is part of the contract callers code against.

## 1. Initial Swagger Setup

1. Check the Go toolchain and module before installing dependencies.
2. Install the CLI and dependency:

   ```sh
   go install github.com/swaggo/swag/cmd/swag@v1.16.6   # pinned; see the Makefile
   go get github.com/swaggo/swag
   ```

3. Choose a Swagger UI adapter compatible with Echo v5. This template uses
   `github.com/swaggo/http-swagger/v2` behind `echo.WrapHandler`. Do not assume an Echo v4 adapter
   is compatible; check the module version before adding it to `go.mod`.
4. Add API metadata to the entrypoint scanned by `swag` — here `cmd/main.go`, above `func main()`:

   ```go
   // @title           IoT Service API
   // @version         1.0.0
   // @description     HTTP API of the iotlab-service. Every endpoint answers with the shared
   // @description     response envelope: `code` carries the domain outcome, `data` the payload.
   // @host            localhost:3000
   // @BasePath        /api/v1
   func main() {
   ```

   `@BasePath` is why `@Router` paths are written without `/api/v1`: the route
   `/api/v1/products/:id` is annotated `@Router /products/{id}`.

5. Generate docs from the repository root. For this template, use a separate output directory
   so generated files do not mix with the existing guidance documentation:

   ```sh
   make swagger    # swag init -g ./cmd/main.go -o docs/swagger --parseInternal --parseDepth 3
   ```

6. Import the generated package with a blank import in the server startup package when needed:

   ```go
   _ "github.com/tdatIT/iotlab-service/docs/swagger"
   ```

7. Register the Swagger UI or static docs route in `internal/router/routes.go`.
   Keep the docs route outside the `/api/v1` group when it should be separate from the API contract.
8. Run `go mod tidy` after the dependency and generated package are stable.

## 2. Add or Update an API

Follow this order:

1. Identify the actual route in `internal/router/routes.go` and its handler.
2. Put annotations directly above the handler method in `internal/handler/<domain>`.
3. Describe the HTTP method, path, tag, summary, request body, response, and errors accurately.
4. Use the actual DTO types from `internal/domain/dtos`; do not create sample schemas only for docs.
5. For path parameters, the `@Param` name must match the route name; for example, `:id` uses `id`.
6. For paginated APIs, document query parameters and the pagination response using the actual DTOs.
7. Regenerate the docs — always through the Makefile, so the flags stay identical to CI and to
   what everyone else runs:

   ```sh
   make swagger
   ```

8. Review the generated file diff; do not edit generated output manually.
9. Stage `docs/swagger/` **in the same commit** as the contract change.

Minimal annotation example:

```go
// CreateProduct handles POST /api/v1/products.
//
//	@Summary		Create a product
//	@Description	Persists a product and emits an `events/product/created` message.
//	@Tags			products
//	@Accept			json
//	@Produce		json
//	@Param			request	body		productdtos.CreateProductReq					true	"Product to create"
//	@Success		200		{object}	response.BaseRes{data=productdtos.ProductDTO}	"Created product"
//	@Failure		400		{object}	response.BaseRes								"Malformed body or failed validation"
//	@Failure		409		{object}	response.BaseRes								"SKU already exists"
//	@Failure		500		{object}	response.BaseRes								"Internal error"
//	@Router			/products [post]
func (h *Handler) CreateProduct(c *echo.Context) error {
```

Two things about that block, both load-bearing:

- The **first line is an ordinary Go doc comment starting with the function name** — required by
  [go-style-core](../go-style-core/SKILL.md) and enforced by the `comment-rules` hook. `// CreateProduct godoc`
  also satisfies it, but the repo writes a real sentence.
- The annotations are **tab-aligned**, the layout `swag fmt` produces. Run `swag fmt` (see `swag fmt --help`
  for the directory flag) after hand-editing, so the block does not fight the next person's formatter.

Every response is rendered into `response.BaseRes` — including errors, which the global handler
(`svcerr.ErrorHandlerEchoFn`) writes as that same envelope. Use `response.BaseRes{data=...}` for the
payload type; there is no `APIResponse` or `ServiceError` type. Adjust status codes to the handler.

## 3. Contract Update Rules

- When changing a route, method, path parameter, request DTO, or response DTO, regenerate Swagger
   in the same change.
- When adding a new API, update its annotations before marking the feature complete.
- Do not document fields, status codes, authentication, or error responses that the handler does not support.
- Do not duplicate validation in annotations; annotations describe the contract, while validation belongs in the DTO/handler.
- Do not add the Swagger route to readiness, liveness, or metrics middleware exceptions without a clear reason.
- Do not commit secrets, tokens, or credentials in Swagger metadata.

## 4. Validation Checklist

- [ ] `make swagger` succeeds from the repository root.
- [ ] `make check-swagger` passes — every business route is documented.
- [ ] `docs/swagger/` is staged in the same commit as the contract change.
- [ ] Every `@Router` path drops the `/api/v1` prefix and uses `{id}`, not `:id`.
- [ ] One `@Failure` per status the handler can actually return.
- [ ] Request, response, parameter and error schemas name real DTOs.
- [ ] `go build ./...` passes.
- [ ] The new API also has its tests — [service-test](../service-test/SKILL.md).

## Available Scripts

- **`scripts/check-swagger.sh`** — S2: every business route in `routes.go` has a matching path+method
  in `docs/swagger/swagger.json`. With `--staged`, S1 as well: a staged contract change must stage a
  regenerated spec. `--json` for machine-readable output. This is what the `swagger-current` hook runs.

## Troubleshooting

- `swag: command not found`: check `go env GOBIN`, `go env GOPATH/bin`, and PATH.
- Internal types are not found: rerun with `--parseInternal`; add `--parseDependency` when a schema is in a dependency.
- Docs do not change after updating annotations: delete `docs/swagger/` and run `make swagger` again.
- Swagger UI import or compile errors: check that the adapter supports Echo v5; do not force the Echo v4 adapter.
- Fields are missing from a schema: confirm that the DTO is referenced directly and its package is scanned by `swag`.
- `make check-swagger` reports a route the spec should have: the `@Router` line is missing, or its path
  still carries `/api/v1`, or `:id` was not rewritten to `{id}`.

## Related Skills

- [go-style-core](../go-style-core/SKILL.md) — Go formatting and style when adding annotations or wiring.
- [go-clean-cqrs-architecture](../go-clean-cqrs-architecture/SKILL.md) — handler, DTO, and route locations in the template.
- [writing-tests-go-projects](../writing-tests-go-projects/SKILL.md) — writing tests for API handlers.
- [service-test](../service-test/SKILL.md) — API coverage and Definition of Done.
