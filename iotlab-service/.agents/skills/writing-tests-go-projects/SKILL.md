---
name: writing-tests-go-projects
description: "Write and fix Go unit tests for this template: black-box (external _test package) tests of command/query handlers driven by Mockery v3 testify mocks, table-driven cases, testify require/assert, and coverage. Use when adding tests, debugging failures, generating mocks, or improving coverage in this repo."
---

# Write Tests for Go Projects

Run tests, write new tests, fix failures, and improve coverage. The default for
this repo is **black-box testing of the application layer** (command/query
handlers) using **Mockery v3 testify mocks** of the repository interface. Read
[resources/mockup.md](./resources/mockup.md) for mock generation and
[resources/test-table.md](./resources/test-table.md) for table-driven tests.

## Workflow

1. Run existing tests first: `go test ./...`
2. Ensure mocks exist for the infra interfaces you depend on:
   `make mocks` (→ `go tool mockery`) or `go generate ./...`. Mocks land in a
   `mock/` sub-package per layer — see [mockup.md](./resources/mockup.md).
3. Add or update tests in `*_test.go` files.
4. Re-run tests and coverage:
   ```
   go test -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out
   ```

## Test placement

- **Black-box (preferred for app/handler layers):** put tests in an external
  `package <pkg>_test`. Import the package under test and the generated
  `mock` package, exercise only the exported API. This is the default for
  command/query handlers.
- **Same-package (white-box):** only for genuinely internal helpers that aren't
  exported. Don't add exported production APIs just to enable testing.

## Black-box handler test (canonical pattern)

A command/query handler depends on `user.Repository`; inject the generated mock
and assert on the mapped service error / returned DTO.

```go
package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/<module>/internal/app/user/command"
	"github.com/<module>/internal/domain/dtos/userdtos"
	"github.com/<module>/internal/domain/models"
	repomock "github.com/<module>/internal/infras/repository/user/mock"
	"github.com/<module>/pkgs/utilities/svcerr"
)

func TestCreateUserCommand_Handle(t *testing.T) {
	ctx := context.Background()

	t.Run("success trims input and returns dto", func(t *testing.T) {
		repo := repomock.NewMockRepository(t)
		repo.EXPECT().
			Create(mock.Anything, mock.MatchedBy(func(u *models.User) bool {
				return u.Name == "Alice" && u.Email == "alice@example.com"
			})).
			Run(func(_ context.Context, u *models.User) { u.ID = 1 }).
			Return(nil).
			Once()

		got, err := command.NewCreateUserCommand(repo).
			Handle(ctx, &userdtos.CreateUserReq{Name: "  Alice  ", Email: " alice@example.com "})

		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, uint(1), got.ID)
	})

	t.Run("empty name rejected before hitting the repo", func(t *testing.T) {
		repo := repomock.NewMockRepository(t)
		got, err := command.NewCreateUserCommand(repo).
			Handle(ctx, &userdtos.CreateUserReq{Name: "  ", Email: "a@b.co"})

		assert.Nil(t, got)
		assert.ErrorIs(t, err, svcerr.ErrInvalidParameters)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("duplicate key maps to already-exists", func(t *testing.T) {
		repo := repomock.NewMockRepository(t)
		repo.EXPECT().Create(mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey).Once()

		_, err := command.NewCreateUserCommand(repo).
			Handle(ctx, &userdtos.CreateUserReq{Name: "Bob", Email: "bob@b.co"})
		assert.ErrorIs(t, err, svcerr.ErrAlreadyExists)
	})

	t.Run("unexpected error maps to internal server", func(t *testing.T) {
		repo := repomock.NewMockRepository(t)
		repo.EXPECT().Create(mock.Anything, mock.Anything).Return(errors.New("db down")).Once()

		_, err := command.NewCreateUserCommand(repo).
			Handle(ctx, &userdtos.CreateUserReq{Name: "Bob", Email: "bob@b.co"})
		assert.ErrorIs(t, err, svcerr.ErrInternalServer)
	})
}
```

## Rules

- Use `require` for assertions that must stop the test (nil checks, fatal
  preconditions); use `assert` for follow-up checks.
- Assert service errors with `errors.Is(err, svcerr.ErrXxx)` — they are pointer
  singletons from `pkgs/utilities/svcerr/common_err.go`.
- Drive the repository with the generated mock's `EXPECT()` builder. Use `Run`
  to simulate side effects on pointer arguments, `mock.MatchedBy` to assert
  inputs, and `AssertNotCalled` for validation/short-circuit paths.
- Prefer one `t.Run` subtest per behavior. Use a **table** only when cases share
  the same setup and just vary inputs/outputs (see
  [test-table.md](./resources/test-table.md)); split cases that need very
  different mock setup into separate tests.
- Keep test doubles out of production packages: generated mocks live in the
  per-layer `mock/` sub-package, never imported by non-test code.
- For repository (`repos_impl.go`) integration tests, prefer a lightweight
  isolated DB (sqlite in-memory or testcontainers) rather than mocking GORM.
- Do not add exported production APIs only to make testing easier.

## Resources

- [Test Table Example](./resources/test-table.md)
- [Mocking with Mockery v3](./resources/mockup.md)
