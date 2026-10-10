---
name: handle-logging
description: "Handles logging in Go projects. Use when the user asks to implement logging, improve log formatting, or add structured logging with libraries like slog or zap."
---

## Handle Logging in Go Projects

Logs should help diagnose production issues. If a log line does not support debugging, monitoring, or incident response,
it is usually noise.

## Logging Libraries

- **slog**: The standard Go structured logger. Use it by default for new projects.
- **zap**: A high-performance structured logger. Use it when the project already depends on it or the user asks for it.

## Structured Logging

Use key-value pairs for dynamic data. Keep the message static.

```go
// Good
slog.Info("order placed", slog.String("order_id", orderID), slog.Float64("total", total))

// Bad
slog.Info(fmt.Sprintf("order %d placed for $%.2f", orderID, total))
```

- Use `snake_case` for field names.
- Prefer typed helpers such as `slog.Int`, `slog.String`, `zap.Int`, and `zap.String`.
- When logging errors with slog, include an `error` field.
- When logging errors with zap, use `zap.Error(err)`.

## Log Levels

Use the lowest level that fits the event.

| Level | When to use                                                 | Default in production |
|-------|-------------------------------------------------------------|-----------------------|
| Debug | Developer diagnostics, internal state, verbose tracing      | Off                   |
| Info  | Lifecycle events such as startup, shutdown, and config load | On                    |
| Warn  | Recoverable problems, retries, deprecated behavior          | On                    |
| Error | Failed operations that need attention                       | On                    |

## Request Logging

- For HTTP servers, log request method, path, status, duration, and user agent at `Info` level.
- Propagate `request_id` to all logs for the same request.

## Log Once

Log an error once at the boundary that can act on it. Lower layers should usually wrap and return the error instead of
logging it again.

```go
// Bad: logged here and again by callers
if err != nil {
    slog.Error("query failed", "err", err)
    return fmt.Errorf("query: %w", err)
}

// Good: wrap and return
if err != nil {
    return fmt.Errorf("query: %w", err)
}
```

HTTP handlers and similar top-level boundaries may log the error and return a sanitized response to the client.

```go
if err != nil {
    slog.Error("checkout failed", "err", err, "user_id", uid)
    http.Error(w, "internal error", http.StatusInternalServerError)
    return
}
```

## What Not to Log

Never log secrets, credentials, PII, or high-cardinality unbounded data.

- Passwords, API keys, tokens, session IDs
- Full credit card numbers, SSNs
- Request or response bodies that may contain user data
- Entire slices or maps with unbounded size