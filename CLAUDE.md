# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

`iotlab-hub` is a plain folder (not a git repo) holding two independent projects. Each has its own `CLAUDE.md`, README,
toolchain and build — **read the sub-project's `CLAUDE.md` before changing code in it**, and run commands from inside
that sub-project.

| Project | Stack | Guidance |
| --- | --- | --- |
| `iotlab-service/` | Go 1.27.1, Echo v5, pgx v5 + TimescaleDB, Paho MQTT, golang-migrate | `iotlab-service/CLAUDE.md`, then `docs/GUIDELINE.md` + `.agents/skills/{go-style-core,go-clean-cqrs-architecture}/SKILL.md` |
| `iotlab-web/` | React 19, Vite, TypeScript strict, react-router v7, Astryx design system, recharts, react-i18next (en/vi) | `iotlab-web/CLAUDE.md` and `iotlab-web/.claude/CLAUDE.md` (Astryx rules + `npx astryx` CLI workflow) |

`iotlab-web/` is its own git repo; `iotlab-service/` is not under git in this checkout.

Deployment: root `docker-compose.yml` (Traefik + TimescaleDB + Mosquitto + iotlab-service built from `./iotlab-service`,
env from `.env`, see `README.md`). The frontend is deployed separately. `iotlab-service/docker-compose.yml` is the older
infra-only file.

## System flow

```
ESP32 (DHT22) ──MQTT──► iotlab-service workers ──► commands ──► repositories ──► TimescaleDB
                                                                                    ▲
iotlab-web (browser) ──HTTP /api/v1──► routes (per-route rate limit) ──► handlers ──► queries
```

MQTT topics: `homelab/iot/env/{device_id}/sensor`, `homelab/iot/devices/{device_id}/telemetry`,
`homelab/iot/devices/{device_id}/status` (retained, `online`/`offline`). Payload formats are in `iotlab-service/README.md`.

## Commands

```bash
# iotlab-service (run inside iotlab-service/)
CONFIG_PATH=config/config-dev make run   # API on :3000, Swagger at /swagger/index.html
make test                                # go test ./...
go test ./internal/app/device/command -run TestRecordSensorCommand_Handle   # single test
make lint | coverage | e2e | mocks | swagger

# iotlab-web (run inside iotlab-web/)
npm run dev      # :5173, calls VITE_API_BASE_URL (default http://localhost:3000/api/v1)
npm run build    # tsc + vite build — the only automated check (no tests/linter)
```

## Cross-project contract

Changes to the HTTP API touch both projects; keep them in sync:

- **Envelope**: every response is `{code, message, data}`, `code === 0` = success. Unwrapped by `iotlab-web/src/lib/api.ts`.
- **DTOs**: `iotlab-service/internal/domain/dtos/devicedtos/` ↔ `iotlab-web/src/features/devices/types.ts` (hand-mirrored
  from Swagger, no codegen). Endpoints ↔ one function each in `iotlab-web/src/features/devices/api.ts`.
- **Swagger**: after changing handler annotations run `make swagger` in the service; the web treats
  `docs/swagger/swagger.json` as its source of truth.
- **No proxy**: the browser hits the API directly, so the service must keep CORS enabled (`internal/http.go`).
  `VITE_API_BASE_URL` is baked in at build time.
- **Rate limits**: each API route has its own per-IP limiter (`routeRateLimit` in service config, default rate 5 /
  burst 10). The web polls every `APP_CONFIG.refreshMs` and retries 429s — adding queries per page can trip the limit.
- There is no `GET /devices/{id}`; the web resolves a device from `GET /devices`.

## Hooks note

`iotlab-service/.claude/settings.json` defines hooks (gofmt on write, comment rules, commit convention, git-add guard,
swagger freshness, test-gate reminder) using `$CLAUDE_PROJECT_DIR`. They only run when Claude Code is started inside
`iotlab-service/`, not from this hub folder — when editing Go here, run `gofmt`/`make lint` yourself.
