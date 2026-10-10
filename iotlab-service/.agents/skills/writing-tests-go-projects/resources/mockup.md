# Mocking with Mockery (v3)

This project generates mocks with **Mockery v3** using the `testify` template.
Mocks are managed as a **Go tool dependency** (no global install) and generated
via `go generate`. Each layer keeps its mocks in a dedicated `mock/` sub-package
next to the interface.

## Standard for this repo

- One mock package **per layer**, in a `mock/` sub-directory of that layer.
- File name: `<interface_snake_case>_mock.go` (e.g. `Repository` → `repository_mock.go`,
  `RedisClient` → `redis_client_mock.go`).
- Package name: `mock`. Struct name: `Mock<InterfaceName>`.
- Mock only **low-level / infrastructure interfaces** (repositories, `orm.ORM`,
  `rdclient.RedisClient`, external adapters) — not application handlers.

## Install (Go tool dependency)

Mockery is tracked in `go.mod` as a tool, so the version is pinned per project
and no global binary is needed (Go 1.24+):

```bash
go get -tool github.com/vektra/mockery/v3@v3
go tool mockery --help        # invoke via the toolchain
```

`testify/mock` needs `objx`; run `go mod tidy` after the first generation if you
see a missing `github.com/stretchr/objx` go.sum entry.

## Configuration — `.mockery.yaml`

```yaml
template: testify
formatter: goimports
all: false

# Default layout inherited by every package below:
# each layer keeps its mocks in a `mock` sub-package, as <name>_mock.go files.
dir: "{{.InterfaceDir}}/mock"
filename: "{{.InterfaceName | snakecase}}_mock.go"
pkgname: "mock"
structname: "Mock{{.InterfaceName}}"

packages:
  github.com/<module>/internal/infras/repository/user:
    config:
      all: true
  github.com/<module>/pkgs/database:
    config:
      all: true
  github.com/<module>/pkgs/rdclient:
    config:
      all: true
```

Add a package path here for every layer whose interfaces you want mocked; set
`all: true` to mock every interface in that package.

## Trigger generation with `go:generate`

A single directive lives in `generate.go` at the module root (a plain
`package tools` file with **no build constraint**, so `go generate ./...` can
discover it):

```go
// generate.go
package tools

//go:generate go tool mockery
```

Regenerate:

```bash
go generate ./...      # works once the tree compiles
# or, the robust bootstrap path (works even on a clean checkout):
make mocks             # → go tool mockery
```

> Gotcha: on a fresh checkout the `mock/` packages don't exist yet, so any test
> that imports them breaks the package graph and `go generate ./...` silently
> does nothing. That's why `make mocks` calls `go tool mockery` directly — it
> regenerates regardless of compile state. Always bootstrap with `make mocks`.

## Generated mock layout (example)

```text
internal/infras/repository/user/mock/repository_mock.go    # package mock, MockRepository
pkgs/database/mock/orm_mock.go                             # package mock, MockORM
pkgs/rdclient/mock/redis_client_mock.go                    # package mock, MockRedisClient
```

## Using a generated mock (testify style)

`New<Mock>(t)` registers cleanup and auto-asserts expectations. Use the typed
`EXPECT()` builder; use `Run` to mutate pointer args (e.g. simulate a DB-assigned
ID), `mock.MatchedBy` to assert on inputs, and `AssertNotCalled` for paths that
must short-circuit before the dependency is touched.

```go
import repomock "github.com/<module>/internal/infras/repository/user/mock"

repo := repomock.NewMockRepository(t)
repo.EXPECT().
	Create(mock.Anything, mock.MatchedBy(func(u *models.User) bool {
		return u.Name == "Alice" && u.Email == "alice@example.com"
	})).
	Run(func(_ context.Context, u *models.User) { u.ID = 1 }). // simulate PK assignment
	Return(nil).
	Once()
```

See [test-table.md](./test-table.md) for full black-box test examples that
consume these mocks.
