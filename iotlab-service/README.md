# iotlab-service

Homelab IoT backend. ESP32 devices (DHT22 sensor) publish readings, health telemetry and online/offline status over MQTT;
the service stores them in TimescaleDB and serves them over a REST API.

```
ESP32 ──MQTT──► workers (internal/worker/event) ──► app commands ──► repositories ──► TimescaleDB
                                                                                         ▲
HTTP client ──► routes (rate limited) ──► handlers ──► app queries ──► repositories ─────┘
```

Stack:

- **[Echo v5](https://echo.labstack.com/)** — HTTP framework, per-route rate limiting
- **[pgx v5](https://github.com/jackc/pgx) + [TimescaleDB](https://www.timescale.com/)** — parameterized SQL over a
  connection pool; hypertables, continuous aggregates, retention policies
- **[golang-migrate](https://github.com/golang-migrate/migrate)** — versioned SQL migrations, embedded in the binary
- **[Paho MQTT](https://github.com/eclipse/paho.mqtt.golang)** — device ingestion
- **`encoding/json/v2`** — JSON for MQTT payloads
- **[Viper](https://github.com/spf13/viper)** — config file + env var override
- **[slog](https://pkg.go.dev/log/slog)** — structured logging

Architecture: clean layered CQRS — Handler/Worker → App (Command/Query) → Repository. See [`CLAUDE.md`](CLAUDE.md) for
the conventions this repo actually follows and [`docs/GUIDELINE.md`](docs/GUIDELINE.md) for the general layer reference.

---

## Requirements

| Tool                    | Version                         |
|-------------------------|---------------------------------|
| Go                      | 1.27.1+                         |
| PostgreSQL + TimescaleDB| TimescaleDB 2.20+ (`tsdb.hypertable` table syntax) |
| MQTT broker             | any (Mosquitto, EMQX)           |
| Docker & Docker Compose | to run the infrastructure       |

---

## MQTT ingestion

The `{device_id}` segment of the topic identifies the device; payloads never carry it. `time` is Unix epoch milliseconds.

| Topic                                         | QoS | Payload                                                                                                                                                                        | Stored in                       |
|-----------------------------------------------|-----|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------|
| `homelab/iot/env/{device_id}/sensor`          | 0   | `{"temp":31.6,"hum":74.5,"time":1791620022386}`                                                                                                                                | `sensor_raw`, `devices.last_seen` |
| `homelab/iot/devices/{device_id}/telemetry`   | 0   | `{"time":…,"fw":"1.2.0-dht22","uptime_s":10443,"reset_reason":"poweron","rssi_dbm":-48,"ip":"192.168.1.156","heap_free":226368,"heap_min":143620,"wifi_reconnects":0,"mqtt_reconnects":0,"sensor_errors":1}` | `device_telemetry`, `devices` (last_seen/fw/ip) |
| `homelab/iot/devices/{device_id}/status` (retained) | 0 | plain text `online` / `offline`                                                                                                                                         | `devices.status`                |

- Devices are registered automatically on their first message.
- Invalid payloads (malformed JSON, duplicate keys, out-of-range temp/humidity, bad IP, unknown status) are logged and dropped.
- A redelivered reading/report (same device + time) is ignored. A status equal to the stored one is a no-op, so
  `status_changed_at` keeps the time of the real transition.

Topics and QoS live under `workers.*` in the config.

---

## HTTP API

Base path `/api/v1`. Every response uses the envelope `{"code": …, "message": …, "data": …}`. Swagger UI: `/swagger`.

| Method | Path                                    | Query                  | Returns                                                   |
|--------|-----------------------------------------|------------------------|-----------------------------------------------------------|
| `GET`  | `/devices`                              |                        | every registered device                                   |
| `GET`  | `/devices/{id}/telemetry/current`       |                        | newest telemetry report (404 when none)                   |
| `GET`  | `/devices/{id}/telemetry/stats`         | `range=15m\|30m\|60m`  | report series + RSSI / heap min/max/avg summary           |
| `GET`  | `/devices/{id}/sensor/current`          |                        | newest temperature/humidity reading (404 when none)       |
| `GET`  | `/devices/{id}/sensor/stats`            | `range=30m\|60m\|90m`  | raw reading series + temperature/humidity summary         |
| `GET`  | `/devices/{id}/sensor/average-24h`      |                        | average temperature/humidity of the last 24h              |
| `GET`  | `/devices/{id}/sensor/stats-24h`        |                        | 30-minute buckets of the last 24h + summary               |

- The 24h endpoints read the `sensor_30m` continuous aggregate and weight each bucket by its sample count, so the
  average equals the raw average without scanning ~5760 rows. Buckets that started before the window are excluded.
- Every API route has its **own** rate limiter (per client IP), configured by `routeRateLimit`. Over the limit → `429`.

Platform endpoints:

| Method | Path         | Description                                       |
|--------|--------------|---------------------------------------------------|
| `GET`  | `/liveness`  | always `200 ok`                                   |
| `GET`  | `/readiness` | `200 ok` / `503 degraded` — checks Postgres + MQTT |
| `GET`  | `/metrics`   | Prometheus metrics                                |

---

## Configuration

| File                     | Use                                                       |
|--------------------------|-----------------------------------------------------------|
| `config/config.yaml`     | defaults (the Docker image loads this one)                |
| `config/config-dev.yaml` | local development profile                                 |

Select the file with `CONFIG_PATH` (path without extension):

```sh
CONFIG_PATH=config/config-dev go run ./cmd/main.go
```

Override any key with an env var — Viper maps `.` → `_`, uppercase:

| Env var                | Config key             | Example                 |
|------------------------|------------------------|-------------------------|
| `DATABASE_HOST`        | `database.host`        | `localhost`             |
| `DATABASE_PORT`        | `database.port`        | `15432`                 |
| `DATABASE_PASSWORD`    | `database.password`    | `secret`                |
| `DATABASE_AUTOMIGRATE` | `database.autoMigrate` | `false`                 |
| `MQTT_BROKER`          | `mqtt.broker`          | `tcp://localhost:1883`  |
| `MQTT_PASSWORD`        | `mqtt.password`        | `secret`                |
| `LOGGER_LEVEL`         | `logger.level`         | `debug`                 |
| `ROUTERATELIMIT_RATE`  | `routeRateLimit.rate`  | `10`                    |

Never commit credentials: keep `password` keys empty in YAML and pass them through env vars.

### Database on a VPS (SSH tunnel)

Postgres on the VPS is bound to `127.0.0.1` only. Forward it to your machine, then point the service at the local port:

```sh
ssh -N -L 15432:localhost:5432 user@vps-ip     # keep this running
```

```powershell
$env:CONFIG_PATH = "config/config-dev"
$env:DATABASE_HOST = "localhost"
$env:DATABASE_PORT = "15432"
$env:DATABASE_PASSWORD = "..."
go run ./cmd/main.go
```

Inside a container use `DATABASE_HOST=host.docker.internal` and forward with `-L 0.0.0.0:15432:localhost:5432`.

### Database migrations

Schema lives in `migrations/` (embedded via `go:embed`) and is applied on startup when `database.autoMigrate` is `true`:

| Migration                         | Content                                                         |
|-----------------------------------|-----------------------------------------------------------------|
| `000001_create_iot_tables`        | extension + `devices`, `sensor_raw`, `device_telemetry`, `device_status_events` |
| `000002_create_sensor_30m`        | `sensor_30m` continuous aggregate                               |
| `000003_create_device_health_1h`  | `device_health_1h` continuous aggregate                         |
| `000004_add_iot_policies`         | refresh + retention policies                                    |

Each continuous aggregate has its own migration: TimescaleDB refuses to create one inside a multi-statement transaction.

> **Pick one schema owner.** If the database was initialised from `init.sql` (e.g. the compose
> `docker-entrypoint-initdb.d` mount), the tables already exist and migration `000001` fails. Either drop that mount and
> let the service migrate, or run with `DATABASE_AUTOMIGRATE=false`.

New migration pair: `make migrate-create name=add_x` (needs the `migrate` CLI) or copy the numbering by hand.

---

## Running

```sh
make run                                   # go run ./cmd/main.go with config/config.yaml
CONFIG_PATH=config/config-dev make run     # dev profile
```

Infrastructure (`docker-compose.yml`): Traefik (TLS via Let's Encrypt), TimescaleDB (`127.0.0.1` only), Mosquitto.

```sh
docker compose up -d timescaledb mosquitto
```

The compose file expects `./db/init.sql` and `./mosquitto/mosquitto.conf`; create them before starting (see the
migration note above about `init.sql`). Compose variables: `DB_USERNAME`, `DB_PASSWORD`, `DB_NAME`, `DB_PORT`,
`MQTT_USERNAME`, `MQTT_PASSWORD`, `MQTT_PORT`, `ACME_EMAIL`, `ACME_CA_SERVER`.

Service image:

```sh
make dockerize                             # docker build -t iotlab-service:local .
docker run -e DATABASE_HOST=… -e DATABASE_PASSWORD=… -e MQTT_BROKER=… iotlab-service:local
```

---

## Project structure

```
cmd/                       entry point (HTTP server + worker group, graceful shutdown)
config/                    AppConfig + config.yaml / config-dev.yaml
migrations/                TimescaleDB schema (golang-migrate, embedded)
internal/
  server.go                dependency wiring
  http.go                  Echo setup (middleware, validator, error handler)
  router/routes.go         routes + per-route rate limiter
  domain/
    models/                table rows (db tags for pgx scanning)
    dtos/devicedtos/       requests (validate tags) and responses
    msgs/                  MQTT payloads
  app/device/              command/ (worker writes) + query/ (API reads)
  handler/device/          HTTP handlers
  worker/event/            MQTT workers: sensor, telemetry, device status
  infras/repository/       device, sensor, telemetry (pgx, parameterized SQL)
pkgs/
  database/                pgx pool, Querier interface, ErrNotFound/ErrDuplicate, migrations
  mqtt/                    Paho wrapper (resubscribes on reconnect)
  probe/                   readiness checks
  logger/                  slog handler
  utilities/               response, svcerr, validate, decorator, paging, mapper
  rdclient/ cache/ locker/ tracing/ httpclient/   kept as libraries, not wired into the service
```

---

## Testing

```sh
make test        # go test ./...
make coverage    # unit tests + 80% coverage gate
make e2e         # e2e suite against real Postgres/MQTT (docker)
make mocks       # regenerate mockery mocks (.mockery.yaml)
make swagger     # regenerate docs/swagger (swag CLI v1.16.6)
```

- App, handler and worker tests use mockery mocks of the repository interfaces.
- Repository tests use [pgxmock](https://github.com/pashagolub/pgxmock): they pin the SQL and argument binding but do
  not execute it — the e2e suite against real TimescaleDB is what proves the queries.

---

## Make targets

```sh
make run | build | test | coverage | e2e | lint | fmt | tidy | mocks | swagger | migrate-create name=x | dockerize | clean
```
