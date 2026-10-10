---
name: go-clean-cqrs-architecture
description: Use when designing or implementing a Go HTTP service with clean architecture + CQRS, matching this repo's template (Echo v5, GORM, slog, generic decorator handlers, svcerr error model, probe health checks, httpclient circuit breaker). Covers layer layout, where interfaces live, command/query handlers, repositories, DTOs, transport wiring, health probes, outbound HTTP clients, structured logging, and the service-error pattern. Use it to scaffold a new domain or a whole new service that looks like this base.
---

# Clean Architecture with CQRS in Go

This skill describes the exact conventions used in this template so an agent can
reproduce the same structure for a new domain or a new service. Follow the
patterns below verbatim — names, layering, and error/logging choices are
intentional. For tests and mocks see
[Writing Tests (Go)](../writing-tests-go-projects/SKILL.md); for logging detail
see [Handle Logging (Go)](../handle-logging/SKILL.md).

## Stack

- HTTP: **Echo v5** (`github.com/labstack/echo/v5`), handlers take `*echo.Context`.
- Persistence: **GORM** (Postgres) behind an `orm.ORM` interface.
- Cache: **go-redis v9** behind a `rdclient.RedisClient` interface.
- Logging: **stdlib `log/slog`** (JSON handler) — NOT zap/zerolog.
- Config: **viper** with `CONFIG_PATH` override and env var mapping (`DATABASE_HOST`, `REDIS_ADDRESS`, …).
- Validation: **go-playground/validator** via Echo's validator hook.
- Health: **`pkgs/probe`** — framework-agnostic `net/http` readiness handler.
- Outbound HTTP: **`pkgs/httpclient`** (resty wrapper) with optional circuit breaker.

## Layers

- **Domain** (`internal/domain`): entities (`models`), request/response DTOs
  (`dtos`), enums/codes (`enums`). Framework-agnostic.
- **Application** (`internal/app/<domain>`): use-case orchestration only —
  command/query handlers. No DB, HTTP, or transport details here.
- **Infrastructure** (`internal/infras`): concrete implementations
  (repositories, adapters). Defines the repository **interface** next to its
  implementation and depends on `pkgs/database` for the `ORM` interface.
- **Interface/Transport** (`internal/handler`, `internal/router`): Echo handlers
  and routes; depend on Application.
- **Shared** (`pkgs`): decorator, logger, db clients, probe, httpclient, utilities.
- **Entry/Config** (`cmd`, `config`, `internal/server.go`, `internal/http.go`).

Group code by domain (e.g. `user`) across layers.

## Project Layout

```
service/
├─ cmd/main.go                            # builds server.NewServer(), starts Echo + workers
├─ config/{config.go,config.yml}          # viper; AppConfig struct
├─ internal/
│  ├─ server.go                           # wiring: config→slog→DB/Redis→probe→repo→app→handler→routes
│  ├─ http.go                             # Echo middleware, validator, global error handler
│  ├─ middleware/tracing.go               # OTel trace middleware
│  ├─ router/routes.go                    # RegisterRoutes(e, handlers..., readyProbe)
│  ├─ app/<domain>/
│  │  ├─ application.go                   # aggregates Command/Query, NewXApplication(repo)
│  │  ├─ command/{create,update,delete}_<domain>.go
│  │  └─ query/{get_*,list_*}.go
│  ├─ domain/
│  │  ├─ models/<domain>.go               # GORM models (gorm tags)
│  │  ├─ dtos/<domain>dtos/<domain>_dto.go# req/res DTOs (validate tags) + NewXDTO mappers
│  │  └─ enums/svc_code.go                # numeric service codes
│  ├─ handler/<domain>/handler.go         # Echo handlers (*echo.Context)
│  └─ infras/
│     ├─ repository/<domain>/
│     │  ├─ repository.go                 # the Repository INTERFACE
│     │  └─ repos_impl.go                 # GORM implementation + NewXRepository(orm.ORM)
│     ├─ adapter/<service>/               # outbound REST adapters (httpclient)
│     └─ mqttpub/                         # MQTT publisher wrapper
├─ pkgs/
│  ├─ utilities/
│  │  ├─ decorator/{command.go,queries.go}# generic CommandHandler/QueryHandler
│  │  ├─ response/                        # BaseRes + SuccessRes
│  │  ├─ svcerr/                          # *Error type, common_err.go, ErrorHandlerEchoFn
│  │  ├─ paging/                          # ListQuery / ListResponse
│  │  ├─ mapper/                          # struct-to-struct helpers
│  │  └─ validate/                        # validator singleton
│  ├─ probe/                              # Checker interface, Probe, DBChecker, RedisChecker
│  ├─ httpclient/                         # resty factory + CBConfig circuit breaker
│  ├─ logger/slog.go                      # NewSlogLogger
│  ├─ database/                           # ORM interface + GORM constructor
│  ├─ rdclient/                           # RedisClient interface + go-redis constructor
│  ├─ cache/                              # Cache interface + Redis-backed impl
│  ├─ locker/                             # Locker interface + Redis SET-NX + Lua release
│  ├─ mqtt/                               # MQTT client wrapper
│  └─ tracing/                            # OTel initialisation
├─ .mockery.yaml                          # see Writing Tests skill
├─ Dockerfile
└─ go.mod
```

## Where Interfaces Live (important)

- The **repository interface** is defined in the infrastructure package next to
  its implementation: `internal/infras/repository/<domain>/repository.go`. The
  application layer imports this interface; the implementation
  (`repos_impl.go`) is wired in `internal/server.go`.
- Low-level infra interfaces (`database.ORM`, `rdclient.RedisClient`) live in
  `pkgs/database/` and `pkgs/rdclient/` next to their constructors.

```go
// internal/infras/repository/user/repository.go
package user

type Repository interface {
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindAndCount(ctx context.Context, limit, offset int) ([]*models.User, int64, error)
	Create(ctx context.Context, user *models.User) error
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
}
```

```go
// internal/infras/repository/user/repos_impl.go
type userReposImpl struct{ db orm.ORM }

func NewUserRepository(database orm.ORM) Repository { return &userReposImpl{db: database} }

func (r *userReposImpl) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.GormDB().WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err // return raw gorm error; the app layer maps it to svcerr
	}
	return &user, nil
}
```

The repository returns **raw GORM errors** (e.g. `gorm.ErrRecordNotFound`,
`gorm.ErrDuplicatedKey`). Mapping to service errors happens in the app layer.

## CQRS via Generic Decorators

`pkgs/utilities/decorator` defines three generic handler interfaces:

```go
type CommandHandler[T any]              interface{ Handle(ctx context.Context, req T) error }
type CommandReturnHandler[T any, E any] interface{ Handle(ctx context.Context, req T) (E, error) }
type QueryHandler[T any, E any]         interface{ Handle(ctx context.Context, req T) (E, error) }
```

Each use case declares a typed alias `I<Name>` over one of these, a private
struct holding only the repository, a `New<Name>` constructor returning the
interface, and a `Handle` method:

```go
// internal/app/user/command/create_user.go
type ICreateUserCommand decorator.CommandReturnHandler[*userdtos.CreateUserReq, *userdtos.UserDTO]

type createUserCommand struct{ repo user.Repository }

func NewCreateUserCommand(repo user.Repository) ICreateUserCommand { return &createUserCommand{repo: repo} }

func (c *createUserCommand) Handle(ctx context.Context, req *userdtos.CreateUserReq) (*userdtos.UserDTO, error) {
	model := &models.User{Name: strings.TrimSpace(req.Name), Email: strings.TrimSpace(req.Email)}
	if err := c.repo.Create(ctx, model); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, svcerr.ErrAlreadyExists
		}
		slog.Error("create user failed", slog.String("error", err.Error()))
		return nil, svcerr.ErrInternalServer
	}
	return userdtos.NewUserDTO(model), nil
}
```

- Commands with no payload back to the caller use `CommandHandler[T]` (delete).
- Commands that return data use `CommandReturnHandler[T, E]` (create/update).
- Reads use `QueryHandler[T, E]`.

Aggregate handlers in `application.go` and inject the repository once:

```go
// internal/app/user/application.go
type Application struct {
	Command commands
	Query   queries
}

func NewUserApplication(userRepo user.Repository) *Application {
	return &Application{
		Command: commands{
			CreateUser: command.NewCreateUserCommand(userRepo),
			UpdateUser: command.NewUpdateUserCommand(userRepo),
			DeleteUser: command.NewDeleteUserCommand(userRepo),
		},
		Query: queries{
			GetUser:   query.NewGetUserByIDQuery(userRepo),
			ListUsers: query.NewListUsersQuery(userRepo),
		},
	}
}
```

## DTOs & Models

- DTOs live in `internal/domain/dtos/<domain>dtos`, carry `json` + `validate`
  tags, and provide `NewXDTO`/`NewXDTOs` mappers from models.
- Models live in `internal/domain/models` with GORM tags.

```go
type CreateUserReq struct {
	Name  string `json:"name" validate:"required,min=2,max=100"`
	Email string `json:"email" validate:"required,email,max=255"`
}
func NewUserDTO(u *models.User) *UserDTO { if u == nil { return nil }; return &UserDTO{ID: u.ID, /* … */} }
```

## Transport (Echo v5 handlers)

Handlers bind+validate, delegate to the app, and return service errors directly
(the global error handler converts them). Success goes through `response`:

```go
func (h *Handler) CreateUser(c *echo.Context) error {
	var req userdtos.CreateUserReq
	if err := c.Bind(&req); err != nil {
		return svcerr.ErrBadRequest
	}
	if err := c.Validate(&req); err != nil {
		return err
	}
	data, err := h.app.Command.CreateUser.Handle(c.Request().Context(), &req)
	if err != nil {
		return err
	}
	success := response.SuccessRes
	success.Data = data
	return success.JSON(c)
}
```

- The `Handler` struct holds `*<domain>.Application`; `NewXHandler(app)`.
- Parse path params with a small helper returning `svcerr.ErrInvalidIDParam`.

**Wrapping a `net/http` handler inside Echo:**

```go
e.GET("/readiness", echo.WrapHandler(readyProbe.Handler()))
```

## Health Checks — Probe

`pkgs/probe` exposes a pure `net/http` readiness handler with no framework dependency.

```go
// Checker interface — implement or use CheckerFunc adapter
type Checker interface {
	Check(ctx context.Context) error
}
```

**Wire in `server.go`:**

```go
readyProbe := probe.New(3 * time.Second).
	Register("postgres", probe.DBChecker(database)).
	Register("redis",    probe.RedisChecker(redisClient))
```

If Redis is unavailable at startup, log a warning and register a static failing checker — do NOT crash the server:

```go
redisClient, redisErr := rdclient.NewRedisClient(cfg)
if redisErr != nil {
	slog.Warn("redis unavailable at startup; readiness probe will report degraded",
		slog.String("error", redisErr.Error()))
}
readyProbe := probe.New(3 * time.Second).Register("postgres", probe.DBChecker(database))
if redisErr == nil {
	readyProbe.Register("redis", probe.RedisChecker(redisClient))
} else {
	readyProbe.Register("redis", probe.CheckerFunc(func(_ context.Context) error { return redisErr }))
}
```

**Register routes:**

```go
e.GET("/liveness",  func(c *echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
})
e.GET("/readiness", echo.WrapHandler(readyProbe.Handler()))
```

**Adding a custom checker for a new infra dependency:**

```go
readyProbe.Register("mqtt", probe.CheckerFunc(func(ctx context.Context) error {
	return mqttCli.Ping(ctx)
}))
```

## Outbound HTTP — httpclient + Circuit Breaker

All outbound REST adapters use `pkgs/httpclient` (resty wrapper). Add
`CircuitBreaker: &httpclient.CBConfig{…}` to protect against downstream failures:

```go
func NewAdapter(cfg *config.HTTPClient) ServiceAdapter {
	c := httpclient.New(httpclient.Config{
		BaseURL:    cfg.BaseURL,
		Timeout:    cfg.Timeout,
		RetryCount: cfg.RetryCount,
		RetryWait:  cfg.RetryWait,
		CircuitBreaker: &httpclient.CBConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			ResetTimeout:     10 * time.Second,
			// Empty Policies uses resty's default (trip on 5xx).
			// Append custom resty.CircuitBreakerPolicy values to trip on 429 etc.
		},
	})
	return &adapter{caller: c}
}
```

States: `Closed → Open → HalfOpen → Closed`. Returns `httpclient.ErrCircuitOpen`
immediately when open. Default `Policies` (empty) uses resty's `CircuitBreaker5xxPolicy`, which trips on HTTP 5xx.

## Errors: the `svcerr` model

- All application/transport errors are `*svcerr.Error` (`pkgs/utilities/svcerr`).
- Use the predefined singletons in `common_err.go`
  (`ErrBadRequest`, `ErrInvalidParameters`, `ErrAlreadyExists`,
  `ErrRecordNotFound`, `ErrInternalServer`, …). Compare with `errors.Is` — they
  are pointer singletons.
- Map infra/GORM errors to a service error at the **app layer**, then log the
  raw cause with slog before returning a generic `ErrInternalServer`.
- The Echo global error handler (`svcerr.ErrorHandlerEchoFn`, registered in
  `internal/http.go`) resolves errors in this order:
    1. `*svcerr.Error` — domain error: use `Status`, `Code`, `Message`.
    2. `validator.ValidationErrors` — 400 + `InvalidArgument` code.
    3. `echo.HTTPStatusCoder` — extract HTTP status via interface; respond with `http.StatusText(code)`.
    4. Default — 500 Internal.
- Add new errors to `common_err.go`; never return raw infra errors from handlers.

```go
// shape of a service error
type Error struct {
	Status               int
	InternalErrorMessage string
	Code                 int    `json:"code"`
	Message              string `json:"message"`
}
```

## Logging

Use stdlib `slog` with typed attributes; never `fmt`-format log messages:

```go
slog.Error("get user failed",
	slog.Uint64("user_id", uint64(req.ID)),
	slog.String("error", err.Error()))
```

The JSON handler is built in `pkgs/logger.NewSlogLogger` and installed as
the default logger in `internal/server.go`. go-redis internal pool logs are
routed through slog at `Debug` level so they respect the configured log level.

## Wiring order (`internal/server.go`)

```
load config
  → set slog default
  → init OTel tracer (if enabled)
  → open ORM (fatal on error)
  → open Redis (warn on error, register failing probe checker)
  → build readyProbe with registered checkers
  → create Echo + newHTTPServer
  → NewXRepository(orm) → NewXApplication(repo) → NewXHandler(app)
  → router.RegisterRoutes(echo, handler, readyProbe)
  → NewGroup → register workers
  → return Server{…}
```

Close DB/Redis/tracer on `Shutdown()`.

## Checklist: add a new domain `foo`

1. `internal/domain/models/foo.go` (GORM model).
2. `internal/domain/dtos/foodtos/foo_dto.go` (req/res + mappers).
3. `internal/infras/repository/foo/{repository.go (interface), repos_impl.go}`.
4. `internal/app/foo/{application.go, command/*, query/*}` using the decorator aliases.
5. `internal/handler/foo/handler.go` (+ `NewFooHandler`).
6. Register routes in `internal/router/routes.go` and wire everything in `internal/server.go`.
7. If new infra dependency: register a checker on `readyProbe` in `server.go`.
8. If outbound HTTP adapter: use `pkgs/httpclient` with `CBConfig`.
9. Add any new errors to `pkgs/utilities/svcerr/common_err.go`.
10. Generate mocks + write black-box tests — see [Writing Tests (Go)](../writing-tests-go-projects/SKILL.md).
