# Development Guideline

This document is the single source of truth for implementing new features in this template.
It covers project structure, layer contracts, conventions, and the exact steps to add a
new business domain to the framework-only scaffold.

---

## Table of Contents

1. [Project Structure](#1-project-structure)
2. [Layer Contracts](#2-layer-contracts)
3. [Domain Layer](#3-domain-layer)
4. [Application Layer — CQRS](#4-application-layer--cqrs)
5. [Handler Layer — HTTP](#5-handler-layer--http)
6. [Worker Layer — MQTT](#6-worker-layer--mqtt)
7. [Infrastructure Layer](#7-infrastructure-layer)
8. [Shared Packages](#8-shared-packages)
9. [Configuration](#9-configuration)
10. [Error Handling](#10-error-handling)
11. [Health Checks — Probe](#11-health-checks--probe)
12. [HTTP Client — Circuit Breaker](#12-http-client--circuit-breaker)
13. [Implementing a New Domain](#13-implementing-a-new-domain)
14. [Checklist — New Feature End-to-End](#14-checklist--new-feature-end-to-end)

---

## 1. Project Structure

```
.
├── cmd/
│   └── main.go                         ← Entry point. Starts HTTP + Worker components.
├── config/
│   ├── config.go                        ← AppConfig struct and all sub-configs.
│   └── config.yml                       ← Default values. Override via env vars.
├── migrations/                          ← Versioned SQL migrations (golang-migrate), embedded via go:embed.
│   ├── embed.go                         ← //go:embed *.sql → migrations.Files.
│   └── NNNNNN_*.{up,down}.sql           ← Migration pairs, applied on startup when DATABASE_AUTOMIGRATE=true.
├── internal/                            ← Everything that must NOT be imported by external packages.
│   ├── server.go                        ← Wires all dependencies. The only place that calls constructors.
│   ├── http.go                          ← Echo instance setup (middleware, validator, error handler).
│   ├── middleware/
│   │   └── tracing.go                   ← OpenTelemetry trace middleware.
│   ├── router/
│   │   └── routes.go                    ← Registers all HTTP routes. One place, no magic.
│   ├── domain/                          ← Pure business types — no framework imports.
│   │   ├── models/                      ← GORM models (DB schema).
│   │   ├── dtos/                        ← Request/Response structs per domain.
│   │   ├── msgs/                        ← MQTT message payload structs.
│   │   └── enums/                       ← Shared constants/error codes.
│   ├── app/                             ← Application logic. One sub-package per domain.
│   │   └── <domain>/
│   │       ├── application.go           ← Aggregates commands + queries for the domain.
│   │       ├── command/                 ← Write operations (create, update, delete).
│   │       └── query/                   ← Read operations (get, list).
│   ├── handler/                         ← HTTP handlers. One sub-package per domain.
│   │   └── <domain>/
│   │       └── handler.go
│   ├── worker/                          ← MQTT subscribers. Transport layer, like handlers.
│   │   ├── worker_group.go              ← Group (Register + StartGroup). Framework code — do not modify.
│   │   └── event/                       ← Child workers, one file per topic.
│   └── infras/                          ← Infrastructure implementations.
│       ├── repository/<domain>/         ← DB access. Implements domain repository interface.
│       ├── adapter/<service>/           ← Outbound REST adapters. Implements domain adapter interface.
│       └── mqttpub/                     ← MQTT publisher. Wraps pkgs/mqtt.
├── pkgs/                                ← Shared, reusable packages. No business logic.
│   ├── httpclient/                      ← Resty HTTP client factory + circuit breaker.
│   ├── probe/                           ← Framework-agnostic health check (net/http).
│   ├── database/                        ← GORM connection factory (interface: ORM).
│   ├── rdclient/                        ← Redis client factory (single/cluster/sentinel).
│   ├── cache/                           ← Cache interface + Redis-backed implementation.
│   ├── locker/                          ← Distributed lock interface + Redis SET-NX implementation.
│   ├── logger/                          ← slog JSON handler.
│   ├── mqtt/                            ← Paho MQTT client wrapper.
│   ├── tracing/                         ← OpenTelemetry tracer initialisation.
│   └── utilities/
│       ├── decorator/                   ← Generic command/query handler interfaces.
│       ├── mapper/                      ← Struct-to-struct and struct-to-map helpers.
│       ├── paging/                      ← Page/size pagination (ListQuery, ListResponse).
│       ├── response/                    ← HTTP response envelope (BaseRes).
│       ├── svcerr/                      ← Typed errors + Echo error handler.
│       └── validate/                    ← go-playground/validator singleton.
├── test/
│   └── e2e/                             ← End-to-end suite, build tag `e2e`. See .agents/skills/service-test.
└── docker/
    └── mosquitto/mosquitto.conf         ← Mosquitto broker config for local dev.
```

---

## 2. Layer Contracts

```
HTTP Request ──► Handler ──► App (Command/Query) ──► Repository / Adapter
MQTT Message ──► Worker  ──► App (Command/Query) ──► Repository / Adapter
```

| Layer                | Responsibility                                           | Allowed dependencies                                                         |
|----------------------|----------------------------------------------------------|------------------------------------------------------------------------------|
| Handler / Worker     | Decode input, validate format, call app, encode response | `app`, `domain/dtos`, `domain/msgs`, `pkgs/utilities`                        |
| App (Command/Query)  | Business logic and orchestration                         | `domain/models`, `domain/dtos`, `infras` interfaces, `pkgs/utilities/svcerr` |
| Repository / Adapter | Data access and external calls                           | `domain/models`, ORM / HTTP client                                           |
| Domain               | Pure types — no behaviour                                | Nothing                                                                      |

**Rules:**

- A lower layer must **never** import a higher layer.
- `infras` implementations are injected via interfaces defined in the domain or app layer.
- Handlers and Workers are parallel entry points — they call the same app layer.

---

## 3. Domain Layer

### Model (`internal/domain/models/`)

GORM model. Maps directly to a DB table.

```go
// internal/domain/models/order.go
package models

import "time"

type Order struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null;index"`
	Status    string `gorm:"size:50;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
```

### DTOs (`internal/domain/dtos/<domain>/`)

Request and response types. Validation tags live here — **only here**.

```go
// internal/domain/dtos/orderdtos/order_dto.go
package orderdtos

type CreateOrderReq struct {
	UserID uint   `json:"user_id" validate:"required"`
	Item   string `json:"item"    validate:"required,min=2"`
}

type OrderDTO struct {
	ID     uint   `json:"id"`
	UserID uint   `json:"user_id"`
	Status string `json:"status"`
}

func NewOrderDTO(m *models.Order) *OrderDTO { ... }
```

### Messages (`internal/domain/msgs/`)

MQTT payload structs. Mirrors DTOs but for message-based transport.

```go
// internal/domain/msgs/order_event_msg.go
package msgs

type OrderEventPayload struct {
	Action  string `json:"action"` // "created" | "cancelled"
	OrderID uint   `json:"order_id"`
}
```

### Enums (`internal/domain/enums/`)

Numeric error codes shared across layers. Add new codes here if needed.
`svcerr.Error.Code` uses these values.

---

## 4. Application Layer — CQRS

Each domain has one `Application` struct that bundles all commands and queries.

### Command (write — returns value)

```go
// internal/app/order/command/create_order.go
package command

type ICreateOrderCommand decorator.CommandReturnHandler[*orderdtos.CreateOrderReq, *orderdtos.OrderDTO]

type createOrderCommand struct{ repo order.Repository }

func NewCreateOrderCommand(repo order.Repository) ICreateOrderCommand {
	return &createOrderCommand{repo: repo}
}

func (c *createOrderCommand) Handle(ctx context.Context, req *orderdtos.CreateOrderReq) (*orderdtos.OrderDTO, error) {
	model := &models.Order{UserID: req.UserID, Status: "pending"}
	if err := c.repo.Create(ctx, model); err != nil {
		return nil, svcerr.ErrInternalServer
	}
	return orderdtos.NewOrderDTO(model), nil
}
```

### Command (write — no return value)

Use `decorator.CommandHandler[T]` instead of `CommandReturnHandler[T, E]`.

### Query (read)

```go
type IGetOrderQuery decorator.QueryHandler[*orderdtos.GetOrderReq, *orderdtos.OrderDTO]
```

### Application aggregator

```go
// internal/app/order/application.go
type Application struct {
Command commands
Query   queries
}

type commands struct {
CreateOrder command.ICreateOrderCommand
}

type queries struct {
GetOrder query.IGetOrderQuery
}

func NewOrderApplication(repo order.Repository) *Application {
return &Application{
Command: commands{CreateOrder: command.NewCreateOrderCommand(repo)},
Query:   queries{GetOrder:    query.NewGetOrderQuery(repo)},
}
}
```

---

## 5. Handler Layer — HTTP

One handler file per domain. Handlers decode, validate, call app, encode response.

```go
// internal/handler/order/handler.go
package order

type Handler struct{ app *orderapp.Application }

func NewOrderHandler(app *orderapp.Application) *Handler { return &Handler{app: app} }

func (h *Handler) CreateOrder(c *echo.Context) error {
	var req orderdtos.CreateOrderReq
	if err := c.Bind(&req); err != nil {
		return svcerr.ErrBadRequest
	}
	if err := c.Validate(&req); err != nil {
		return err
	}

	data, err := h.app.Command.CreateOrder.Handle(c.Request().Context(), &req)
	if err != nil {
		return err
	}

	res := response.SuccessRes
	res.Data = data
	return res.JSON(c)
}
```

**Register in router:**

```go
// internal/router/routes.go
orderRoute := v1.Group("/orders")
orderRoute.POST("", orderHandler.CreateOrder)
orderRoute.GET("/:id", orderHandler.GetOrder)
```

**Wrapping a standard `net/http` handler inside Echo:**

```go
e.GET("/readiness", echo.WrapHandler(someHttpHandler))
```

**Parsing helpers (keep in handler file):**

```go
func parseIDParam(c *echo.Context) (uint, error) {
id, err := strconv.ParseUint(c.Param("id"), 10, 64)
if err != nil || id == 0 {
return 0, svcerr.ErrInvalidIDParam
}
return uint(id), nil
}
```

---

## 6. Worker Layer — MQTT

Workers are the MQTT equivalent of HTTP handlers — same app layer, different transport.

### Child worker (`internal/worker/event/`)

```go
// internal/worker/event/order_event_worker.go
package event

type OrderEventWorker struct {
	client mqttclient.Client
	app    *orderapp.Application
	cfg    *config.WorkerConfig
}

func NewOrderEventWorker(client mqttclient.Client, app *orderapp.Application, cfg *config.WorkerConfig) *OrderEventWorker {
	return &OrderEventWorker{client: client, app: app, cfg: cfg}
}

func (w *OrderEventWorker) Start(ctx context.Context) error {
	err := w.client.Subscribe(w.cfg.Topic, w.cfg.QoS, func(topic string, payload []byte) {
		go w.handle(ctx, payload) // non-blocking: never hold the MQTT message loop
	})
	if err != nil {
		return fmt.Errorf("order event worker: subscribe %s: %w", w.cfg.Topic, err)
	}
	<-ctx.Done()
	return nil
}

func (w *OrderEventWorker) handle(ctx context.Context, raw []byte) {
	var event msgs.OrderEventPayload
	if err := json.Unmarshal(raw, &event); err != nil {
		slog.Error("order event: malformed payload", slog.String("error", err.Error()))
		return
	}
	switch event.Action {
	case "cancel":
		// call w.app.Command.CancelOrder.Handle(ctx, &req)
	}
}
```

### Register in server.go

```go
workerGroup.Register(
event.NewOrderEventWorker(mqttCli, orderApplication, &cfg.Workers.OrderEvent),
)
```

### Config

```yaml
# config/config.yml
workers:
  orderEvent:
    topic: "events/order"
    qos: 1
```

```go
// config/config.go
type Workers struct {
OrderEvent WorkerConfig
}
```

---

## 7. Infrastructure Layer

### Repository (`internal/infras/repository/<domain>/`)

Two files: interface + implementation.

**Interface:**

```go
// internal/infras/repository/order/repository.go
package order

type Repository interface {
	FindByID(ctx context.Context, id uint) (*models.Order, error)
	Create(ctx context.Context, order *models.Order) error
}
```

**Implementation** uses `orm.ORM`:

```go
// internal/infras/repository/order/repos_impl.go
type orderRepository struct{ orm orm.ORM }

func NewOrderRepository(o orm.ORM) Repository { return &orderRepository{orm: o} }

func (r *orderRepository) FindByID(ctx context.Context, id uint) (*models.Order, error) {
var order models.Order
if err := r.orm.GormDB().WithContext(ctx).First(&order, id).Error; err != nil {
return nil, err
}
return &order, nil
}
```

**Generate mock** for unit tests:

Mocks are configured **per package** in `.mockery.yaml` (not via `//go:generate`
directives). Add the new repository package under `packages:` and run:

```bash
go tool mockery
```

Mocks land in `<package>/mock/<interface>_mock.go` with package name `mock`
and struct type `Mock<Interface>`.

### Outbound Adapter (`internal/infras/adapter/<service>/`)

Interface defined in the package, implementation wires `pkgs/httpclient`:

```go
// adapter.go — interface
type ServiceAdapter interface {
GetData(ctx context.Context, req *dto.Req) (*dto.Resp, error)
}

// adapter_impl.go — implementation
func NewAdapter(cfg *config.HTTPClient) ServiceAdapter {
c := httpclient.New(httpclient.Config{BaseURL: cfg.BaseURL, ...})
return &adapter{caller: c}
}
```

### MQTT Publisher (`internal/infras/mqttpub/`)

Wraps `pkgs/mqtt.Client`. Inject `Publisher` into app commands that need to emit events.

```go
pub.Publish(ctx, "events/order", payload) // topic, qos=1, retained=false
```

---

## 8. Shared Packages

These packages contain zero business logic. Do not modify unless fixing a bug.

| Package                    | Use when                                                                           |
|----------------------------|------------------------------------------------------------------------------------|
| `pkgs/utilities/svcerr`    | Returning or wrapping errors from handlers/app                                     |
| `pkgs/utilities/response`  | Building HTTP responses: `res := response.SuccessRes; res.Data = dto; res.JSON(c)` |
| `pkgs/utilities/paging`    | Paginated list endpoints: `ListQuery` (input), `ListResponse` (output)             |
| `pkgs/utilities/mapper`    | Copying structs or converting struct → map                                         |
| `pkgs/utilities/validate`  | Already wired via `e.Validator = validate.GetValidator()` — no direct use needed   |
| `pkgs/utilities/decorator` | Generic `CommandHandler[T]` / `QueryHandler[T,E]` interfaces                       |
| `pkgs/httpclient`          | Building outbound HTTP adapters; optionally add circuit breaker via `CBConfig`     |
| `pkgs/probe`               | Registering health checkers and exposing a `net/http` readiness handler            |
| `pkgs/database`            | GORM connection factory; exposes the `ORM` interface consumed by repositories      |
| `pkgs/rdclient`            | Redis client factory (single/cluster/sentinel); exposes `RedisClient`              |
| `pkgs/cache`               | Read-through / write-through caching. `Cache` interface + Redis-backed impl        |
| `pkgs/locker`              | Distributed mutual exclusion. `Locker` interface + Redis SET-NX + Lua release      |
| `pkgs/mqtt`                | Direct MQTT pub/sub — used by `mqttpub` and workers, not by app layer              |
| `pkgs/logger`              | Only used in `server.go` to set the default `slog` handler                         |
| `pkgs/tracing`             | Only used in `server.go` to initialise the OTel tracer                             |

### Cache and Locker — Redis-backed helpers

`pkgs/cache` and `pkgs/locker` are two thin interfaces built on top of
`pkgs/rdclient`, kept separate so the app layer never touches the raw Redis
client. Both are safe to inject as constructor arguments.

```go
// Cache — read-through pattern
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, bool)
    Set(ctx context.Context, key string, value []byte) error
    SetWithTTL(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}

// Locker — idempotency / anti-double-submit lock
type Locker interface {
    Acquire(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error)
    Release(ctx context.Context, key string, token string) error
}
```

The Redis implementations live in `pkgs/cache/redis_impl.go` and
`pkgs/locker/redis_impl.go`. `Acquire` uses `SET key <uuid> NX PX ttl`; `Release`
runs a Lua script that only deletes the key when the token still matches — so
an expired lock cannot be released by the holder that lost it.

```go
appCache  := cache.NewRedisCache(redisClient.Client())
appLocker := locker.NewRedisLocker(redisClient.Client())
```

Common uses in this template:

- **Cache** — a short-TTL read-through for hot GET endpoints (e.g. one order,
  one product). Invalidate on write in the same command handler.
- **Locker** — wrap a write path that must not run concurrently for the same
  business key: payment, refund, external-call fan-out. On `ok == false`
  return `svcerr.ErrTooManyRequests` (or map to your own idempotent-retry
  response).

---

## 9. Configuration

### Adding a config key

1. Add the field to the relevant struct in `config/config.go`.
2. Add the default value to `config/config.yml`.
3. Document the env override as a comment: `# override: SECTION_FIELD | required`, or `# optional`
   when the field may stay empty.

**No other comments in the YAML.** Never explain business logic, behaviour, or what a field does
inside `config/*.yaml` — that documentation lives here.

Viper maps YAML keys to env vars with the rule: `.` → `_`, all uppercase.
Example: `adapters.product.apiKey` → `ADAPTERS_PRODUCT_APIKEY`.

### Sensitive fields

Never commit real credentials. Leave the YAML value empty and document the env var:

```yaml
database:
  password: ""   # override: DATABASE_PASSWORD
```

### Rate limiter

The Echo rate-limit middleware is wired in `internal/http.go` and reads
`cfg.RateLimit`. It is **disabled by default** so unit tests and local
development are not gated on it. Enable it before running load tests or
exposing the service publicly:

```yaml
rateLimit:
  enabled: true
  rate: 500          # requests per second per client
  burst: 1000        # token bucket depth
  expiresIn: "1m"    # idle bucket eviction
```

### Database migrations

Schema changes are managed with [golang-migrate](https://github.com/golang-migrate/migrate),
**not** GORM `AutoMigrate`. Migration files live in `migrations/` and are embedded into the binary
via `//go:embed`, so the service is self-contained (no directory to ship alongside the image).

- **Auto-run on startup**: `internal/server.go` calls `database.RunMigrations` when
  `cfg.Database.AutoMigrate` is true. It reuses the shared `*sql.DB` (`ORM.SQLDB()`), so it inherits
  the connection's `search_path`; golang-migrate tracks applied versions in a `schema_migrations`
  table. Applying up-to-date schema is a no-op.

  ```yaml
  database:
    autoMigrate: true   # override: DATABASE_AUTOMIGRATE
  ```

  Default `true` so local dev works out of the box. In production, set `DATABASE_AUTOMIGRATE=false`
  when migrations are applied out-of-band (CI/CD, ops job) rather than at boot.

- **Authoring a migration**: create a `NNNNNN_<description>.up.sql` / `.down.sql` pair under
  `migrations/`. Use the `migrate` CLI if installed — `make migrate-create name=add_orders` — or add
  the files by hand following the existing numbering. They are picked up automatically on the next
  startup (or by disabling auto-run and applying them with the CLI).

### Docker / container overrides

When running inside Docker Compose, services communicate by container name, not `localhost`.
Pass env vars in `docker-compose.yml` to override the YAML defaults:

```yaml
environment:
  - DATABASE_HOST=postgres
  - REDIS_ADDRESS=redis:6379
  - MQTT_BROKER=tcp://emqx:1883
```

`REDIS_ADDRESS` overrides the `redis.address` slice — Viper's `WeaklyTypedInput` coerces the string into
`[]string{"redis:6379"}`.

---

## 10. Error Handling

### Predefined errors (`pkgs/utilities/svcerr/common_err.go`)

Use existing errors before creating new ones:

| Error var              | HTTP | Use case                                       |
|------------------------|------|------------------------------------------------|
| `ErrBadRequest`        | 400  | Bind/parse failure                             |
| `ErrInvalidParameters` | 400  | Malformed query params                         |
| `ErrInvalidIDParam`    | 400  | Non-numeric path `/:id`                        |
| `ErrUnauthenticated`   | 401  | Missing/invalid token                          |
| `ErrPermissionDenied`  | 403  | Insufficient rights                            |
| `ErrNotFound`          | 404  | Generic not found                              |
| `ErrRecordNotFound`    | 404  | DB record missing                              |
| `ErrAlreadyExists`     | 409  | Unique constraint violation                    |
| `ErrInternalServer`    | 500  | Unexpected error — always log before returning |

### Adding a new error

```go
// pkgs/utilities/svcerr/common_err.go
ErrQuotaExceeded = &Error{
Status:  402,
Code:    enums.ResourceExhausted,
Message: "Quota exceeded",
}
```

### Error handler (`pkgs/utilities/svcerr/handle_err_fn.go`)

Registered as `e.HTTPErrorHandler` in `internal/http.go`. Resolution order:

1. `*svcerr.Error` — domain error: use `Status`, `Code`, `Message` directly.
2. `validator.ValidationErrors` — 400 + `enums.InvalidArgument`.
3. `echo.HTTPStatusCoder` — extract HTTP status via interface; respond with `http.StatusText(code)`.
4. Default — 500 + `enums.Internal` + `"internal server error"`.

HEAD requests always respond with `NoContent(code)` (no body).

### Logging before returning 500

```go
slog.Error("create order failed", slog.String("error", err.Error()))
return nil, svcerr.ErrInternalServer
```

Never expose raw error messages to the caller. Log them, return a typed error.

---

## 11. Health Checks — Probe

Health check logic lives in `pkgs/probe/` — a pure `net/http` package with **no framework dependency**.

### Probe package (`pkgs/probe/`)

| File          | Contents                                                                                 |
|---------------|------------------------------------------------------------------------------------------|
| `probe.go`    | `Checker` interface, `CheckerFunc` adapter, `Probe` struct, `Handler() http.HandlerFunc` |
| `checkers.go` | `DBChecker(orm.ORM)`, `RedisChecker(rdclient.RedisClient)` — built-in adapter functions  |

### Wire in `server.go`

```go
readyProbe := probe.New(3 * time.Second).
Register("postgres", probe.DBChecker(database)).
Register("redis", probe.RedisChecker(redisClient))
```

### Register routes

```go
// internal/router/routes.go
e.GET("/liveness", func (c *echo.Context) error {
return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
})
e.GET("/readiness", echo.WrapHandler(readyProbe.Handler()))
```

`Handler()` runs all registered checkers **concurrently** within the request context timeout.
Responds `200 ok` when all pass, `503 degraded` with per-component details when any fail.

### Adding a custom checker

Implement `probe.Checker` or use `probe.CheckerFunc`:

```go
readyProbe.Register("mqtt", probe.CheckerFunc(func(ctx context.Context) error {
return mqttCli.Ping(ctx)
}))
```

---

## 12. HTTP Client — Circuit Breaker

`pkgs/httpclient` wraps [go-resty](https://github.com/go-resty/resty) and optionally adds a circuit breaker at the
`http.RoundTripper` level — transparent to all resty methods.

### Config

```go
type Config struct {
BaseURL        string
Timeout        time.Duration
KeepAlive      time.Duration
RetryCount     int
RetryWait      time.Duration
RetryCondition resty.RetryConditionFunc
Debug          bool
CircuitBreaker *CBConfig // nil disables the circuit breaker
}

type CBConfig struct {
FailureThreshold uint64        // failures counted before the circuit opens
SuccessThreshold uint64        // consecutive half-open successes to close
ResetTimeout     time.Duration // how long the circuit stays open before probing
Policies         []resty.CircuitBreakerPolicy // empty = default 5xx policy
}
```

### State machine

```
Closed ──(FailureThreshold failures)──► Open ──(ResetTimeout elapsed)──► HalfOpen
  ▲                                                                          │
  └───────────────(SuccessThreshold successes)──────────────────────────────┘
```

Any failure in `HalfOpen` immediately returns to `Open`. `Policies` defaults to
`resty.CircuitBreaker5xxPolicy` (trip on HTTP 5xx). Add extra policies to trip
on 429s or bespoke conditions.

### Usage in an adapter

```go
func NewAdapter(cfg *config.HTTPClient) ServiceAdapter {
c := httpclient.New(httpclient.Config{
BaseURL: cfg.BaseURL,
Timeout: cfg.Timeout,
CircuitBreaker: &httpclient.CBConfig{
FailureThreshold: 5,
SuccessThreshold: 2,
ResetTimeout:     10 * time.Second,
// Empty Policies uses the default (trip on 5xx). Append resty policies to
// also trip on 429, network errors, etc.
},
})
return &adapter{caller: c}
}
```

When the circuit is open, `MakeRequest()` returns `httpclient.ErrCircuitOpen` immediately without hitting the network.

---

## 13. Implementing a New Domain

### Step 1 — Define the domain types

```
internal/domain/models/<domain>.go          ← GORM model
internal/domain/dtos/<domain>dtos/          ← request/response types with validate tags
internal/domain/msgs/<domain>_event_msg.go  ← MQTT payload (only if worker needed)
```

### Step 2 — Define the repository interface

```
internal/infras/repository/<domain>/repository.go
```

Register the package under `packages:` in `.mockery.yaml` and run `go tool mockery` to generate test mocks.

### Step 3 — Implement the repository

```
internal/infras/repository/<domain>/repos_impl.go
```

### Step 4 — Write application commands and queries

```
internal/app/<domain>/command/<action>.go   ← one file per command
internal/app/<domain>/query/<action>.go     ← one file per query
internal/app/<domain>/application.go        ← aggregates all commands/queries
```

### Step 5 — Write the HTTP handler

```
internal/handler/<domain>/handler.go
```

### Step 6 — Register routes

```go
// internal/router/routes.go
domainRoute := v1.Group("/<domain>s")
domainRoute.POST("", domainHandler.Create<Domain>)
```

### Step 7 — Wire in server.go

```go
// internal/server.go — inside NewServer()
<domain>Repo := <domain>repos.New<Domain>Repository(database)
<domain>Application := <domain>app.New<Domain>Application(<domain>Repo)
<domain>Handle := <domain>handler.New<Domain>Handler(<domain>Application)
router.RegisterRoutes(echoApp, ..., <domain>Handle)
```

### Step 8 — (Optional) Add a worker

```
internal/worker/event/<domain>_event_worker.go
```

Add config to `config/config.go` (`Workers.<Domain>Event WorkerConfig`) and `config/config.yml`.

Register in server.go:

```go
workerGroup.Register(
event.New<Domain>EventWorker(mqttCli, <domain>Application, &cfg.Workers.<Domain>Event),
)
```

### Step 9 — (Optional) Add an outbound adapter

```
internal/infras/adapter/<service>/adapter.go
internal/infras/adapter/<service>/adapter_impl.go
internal/infras/adapter/<service>/dto/
```

Add HTTP client config to `config/config.go` (`Adapters.<Service> HTTPClient`) and YAML.

### Step 10 — (Optional) Add a health checker

If the new infra dependency should be reflected in `/readiness`, register a checker in `server.go`:

```go
readyProbe.Register("<name>", probe.CheckerFunc(func(ctx context.Context) error {
return <client>.Ping(ctx)
}))
```

---

## 14. Checklist — New Feature End-to-End

**Domain:**

- [ ] Model created and GORM-tagged
- [ ] DTOs created — validate tags on all user-facing fields
- [ ] `NewXxxDTO(model)` constructor handles nil input

**Application:**

- [ ] Command/Query interface declared with `decorator.*Handler`
- [ ] Business logic in `Handle()` — no nil/zero re-checks (handler validated already)
- [ ] Infra errors mapped to `svcerr.*` — raw errors never returned
- [ ] Unexpected errors logged before returning `svcerr.ErrInternalServer`

**Handler:**

- [ ] `c.Bind` → on error return `svcerr.ErrBadRequest`
- [ ] `c.Validate` → on error return `err` as-is
- [ ] Success path: `res := response.SuccessRes; res.Data = dto; return res.JSON(c)`
- [ ] Route registered in `internal/router/routes.go`

**Worker (if applicable):**

- [ ] Implements `worker.Worker` interface (`Start(ctx) error`)
- [ ] Handler dispatched via `go w.handle(ctx, payload)` — MQTT loop never blocked
- [ ] Topic and QoS read from `config.WorkerConfig`, not hardcoded
- [ ] Registered in `server.go` via `workerGroup.Register(...)`

**Infrastructure:**

- [ ] Repository interface in `repository/<domain>/repository.go`
- [ ] Mock generated (`go generate`) for unit tests
- [ ] No validation in repository — trusts the app layer
- [ ] Outbound adapter uses `pkgs/httpclient`; consider `CBConfig` for circuit breaker

**Probe (if new infra dependency):**

- [ ] Checker registered in `server.go` via `readyProbe.Register(...)`

**Config:**

- [ ] New keys added to `config/config.go` struct
- [ ] Defaults set in `config/config.yml`
- [ ] Sensitive fields empty in YAML with `# override: ENV_VAR` comment
- [ ] Docker Compose env overrides added if service name differs from `localhost`

**Tests:**

- [ ] App layer: unit tests with mock repository
- [ ] Handler layer: unit tests with fake repo + `newJSONContext`
- [ ] Repository layer: integration tests against a real Postgres (docker-compose or testcontainers) or an in-memory driver of your choice
- [ ] E2E: `test/e2e/<domain>_api_test.go` with a `// @e2e METHOD PATH` annotation per API in scope
- [ ] Both gates pass: `make coverage` (unit ≥ 80%) and `make e2e` (every selected API has an e2e test)

---

## Sample Domain — `product`

This template ships with a **sample `product` domain** so you have a working
reference to copy from when building your own. It exercises the full CQRS
layout end-to-end plus two power features:

- **Read-through caching** — `GET /api/v1/products/:id` reads through
  `pkgs/cache` (Redis) with a 30-second TTL; writes invalidate the key.
- **Outbound events** — every create/update/delete publishes to
  `events/product/{created,updated,deleted}` via `mqttpub`.

Sample endpoints:

| Method   | Path                   | Purpose                                    |
|----------|------------------------|--------------------------------------------|
| `POST`   | `/api/v1/products`     | Create a product; emits `product.created`  |
| `GET`    | `/api/v1/products`     | Paginated list                             |
| `GET`    | `/api/v1/products/:id` | Read (cached, 30s)                         |
| `PUT`    | `/api/v1/products/:id` | Patch fields; invalidates cache            |
| `DELETE` | `/api/v1/products/:id` | Delete; invalidates cache                  |

Sample files (delete these together when you don't want the sample):

```
internal/domain/models/product.go
internal/domain/dtos/productdtos/
internal/domain/msgs/product_event_msg.go
internal/app/product/
internal/handler/product/
internal/infras/repository/product/
migrations/000001_create_products_table.up.sql
migrations/000001_create_products_table.down.sql
```

Also strip the sample from these files:

- `internal/server.go` — the `Sample: …` Product domain wiring block, plus the
  `productHandle` argument passed to `router.RegisterRoutes`. (The migration
  runner just above it is framework code — keep it; only delete the sample
  `products` migration files listed above.)
- `internal/router/routes.go` — the `productRoute` block; then adjust
  `RegisterRoutes` to drop its `*product.Handler` parameter.
- `.mockery.yaml` — the `internal/infras/repository/product` entry.

Everything under `internal/app/`, `internal/handler/`,
`internal/infras/repository/`, `internal/infras/adapter/`, and
`internal/worker/event/` is domain code and is owned by the team building the
service. Follow [Section 13](#13-implementing-a-new-domain) to add your own.
