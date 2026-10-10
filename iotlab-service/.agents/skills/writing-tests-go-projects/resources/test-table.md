# Test Tables

Use table-driven tests with [subtests] when test logic is the same and only
inputs/expected outputs change.

[subtests]: https://go.dev/blog/subtests

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td>

```go
// func TestSplitHostPort(t *testing.T)

host, port, err := net.SplitHostPort("192.0.2.0:8000")
require.NoError(t, err)
assert.Equal(t, "192.0.2.0", host)
assert.Equal(t, "8000", port)

host, port, err = net.SplitHostPort("192.0.2.0:http")
require.NoError(t, err)
assert.Equal(t, "192.0.2.0", host)
assert.Equal(t, "http", port)

host, port, err = net.SplitHostPort(":8000")
require.NoError(t, err)
assert.Equal(t, "", host)
assert.Equal(t, "8000", port)

host, port, err = net.SplitHostPort("1:8")
require.NoError(t, err)
assert.Equal(t, "1", host)
assert.Equal(t, "8", port)
```

</td><td>

```go
// func TestSplitHostPort(t *testing.T)

tests := []struct{
  give     string
  wantHost string
  wantPort string
}{
  {
    give:     "192.0.2.0:8000",
    wantHost: "192.0.2.0",
    wantPort: "8000",
  },
  {
    give:     "192.0.2.0:http",
    wantHost: "192.0.2.0",
    wantPort: "http",
  },
  {
    give:     ":8000",
    wantHost: "",
    wantPort: "8000",
  },
  {
    give:     "1:8",
    wantHost: "1",
    wantPort: "8",
  },
}

for _, tt := range tests {
  t.Run(tt.give, func(t *testing.T) {
    host, port, err := net.SplitHostPort(tt.give)
    require.NoError(t, err)
    assert.Equal(t, tt.wantHost, host)
    assert.Equal(t, tt.wantPort, port)
  })
}
```

</td></tr>
</tbody></table>

Test tables reduce duplication and make adding new cases easier.

Recommended naming:

- `tests` for the test case slice
- `tt` for each test case
- `give*` and `want*` fields for inputs and expectations

```go
tests := []struct{
  give     string
  wantHost string
  wantPort string
}{
  // ...
}

for _, tt := range tests {
  // ...
}
```

## Avoid Unnecessary Complexity in Table Tests

Avoid large tables with many conditional branches or mock switches. If each case
needs very different setup/assertions, split into separate tests or smaller
tables.

Good guardrails:

- Keep each table focused on one behavior.
- Keep the subtest body short.
- Avoid many `if` branches inside the loop.
- Ensure all fields are used by all cases.

A single `shouldErr` branch is acceptable when the body stays simple.

<table>
<thead><tr><th>Bad</th><th>Good</th></tr></thead>
<tbody>
<tr><td>

```go
func TestComplicatedTable(t *testing.T) {
  tests := []struct {
    give          string
    want          string
    wantErr       error
    shouldCallX   bool
    shouldCallY   bool
    giveXResponse string
    giveXErr      error
    giveYResponse string
    giveYErr      error
  }{
    // ...
  }

  for _, tt := range tests {
    t.Run(tt.give, func(t *testing.T) {
      // setup mocks (Mockery v3 / testify)
      xMock := xmock.NewMockX(t)
      if tt.shouldCallX {
        xMock.EXPECT().Call().Return(tt.giveXResponse, tt.giveXErr)
      }
      yMock := ymock.NewMockY(t)
      if tt.shouldCallY {
        yMock.EXPECT().Call().Return(tt.giveYResponse, tt.giveYErr)
      }

      got, err := DoComplexThing(tt.give, xMock, yMock)

      // verify results
      if tt.wantErr != nil {
        require.ErrorIs(t, err, tt.wantErr)
        return
      }
      require.NoError(t, err)
      assert.Equal(t, tt.want, got)
    })
  }
}
```

</td><td>

```go
func TestShouldCallX(t *testing.T) {
  // setup mocks (Mockery v3 / testify) — NewMockX(t) auto-asserts on cleanup
  xMock := xmock.NewMockX(t)
  xMock.EXPECT().Call().Return("XResponse", nil)

  yMock := ymock.NewMockY(t)

  got, err := DoComplexThing("inputX", xMock, yMock)

  require.NoError(t, err)
  assert.Equal(t, "want", got)
}

func TestShouldCallYAndFail(t *testing.T) {
  // setup mocks
  xMock := xmock.NewMockX(t)

  yMock := ymock.NewMockY(t)
  yMock.EXPECT().Call().Return("YResponse", nil)

  _, err := DoComplexThing("inputY", xMock, yMock)
  assert.ErrorIs(t, err, errFailedY)
}
```

</td></tr>
</tbody></table>

This complexity makes it more difficult to change, understand, and prove the
correctness of the test.

Prefer readability over strict style rules.

## Parallel Tests

Parallel tests, like some specialized loops (for example, those that spawn
goroutines or capture references as part of the loop body),
must take care to explicitly assign loop variables within the loop's scope to
ensure that they hold the expected values.

```go
tests := []struct{
  give string
  // ...
}{
  // ...
}

for _, tt := range tests {
  tt := tt
  t.Run(tt.give, func(t *testing.T) {
    t.Parallel()
    // ...
  })
}
```

Use the `tt := tt` shadowing pattern in parallel subtests to avoid loop-variable
capture issues across Go versions and code styles.