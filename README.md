# iotlab-hub

Homelab IoT stack. ESP32 devices (DHT22) publish readings over MQTT; `iotlab-service` stores them in TimescaleDB and
serves a REST API behind Traefik.

| Folder | Content |
| --- | --- |
| `iotlab-service/` | Go backend (MQTT workers + REST API) — see its [README](iotlab-service/README.md) |
| `iotlab-web/` | React frontend — deployed separately (Vercel / its own Docker image), not part of this compose |
| `docker-compose.yml` | Traefik + TimescaleDB + Mosquitto + iotlab-service |
| `mosquitto/mosquitto.conf` | broker config (auth required, persistence on) |

```
ESP32 ──MQTT :1883──► mosquitto ──► iotlab-service ──► timescaledb
Browser ──HTTPS :443──► traefik ──► iotlab-service :3000
```

## Requirements

- Docker + Docker Compose v2 (or Podman with a Docker-compatible socket)
- For HTTPS: a domain whose DNS points to the host, and ports 80/443 reachable from the internet

## Setup

```sh
cp .env.example .env
# edit .env: API_DOMAIN, ACME_EMAIL, DB_PASSWORD, MQTT_PASSWORD
docker compose up -d --build
docker compose ps                     # iotlab-service should become "healthy"
docker compose logs -f iotlab-service
```

`DB_PASSWORD`, `MQTT_PASSWORD` and `API_DOMAIN` are required; compose refuses to start without them.

On first start the service applies its migrations (`DATABASE_AUTOMIGRATE=true`), so the database needs no init script.

### TLS certificates

Traefik uses the Let's Encrypt **staging** CA by default, so browsers will warn about the certificate. Once
`https://<API_DOMAIN>/liveness` answers, switch to production:

```sh
# .env
ACME_CA_SERVER=https://acme-v02.api.letsencrypt.org/directory
```

```sh
docker compose up -d traefik
```

If certificates were already issued by staging, remove them first: `docker compose down traefik && docker volume rm iotlab_traefik-acme`.

## Endpoints

| URL | Description |
| --- | --- |
| `https://<API_DOMAIN>/api/v1/...` | REST API (see `iotlab-service/README.md`) |
| `https://<API_DOMAIN>/swagger` | Swagger UI |
| `https://<API_DOMAIN>/liveness`, `/readiness` | health checks (readiness checks Postgres + MQTT) |
| `https://<API_DOMAIN>/metrics` | Prometheus metrics |
| `http://127.0.0.1:3000` | API directly on the host, bypassing Traefik |
| `tcp://<host>:1883` | MQTT broker for devices (user/password from `.env`) |
| `127.0.0.1:5432` | Postgres, host-local only |

Point the frontend at the API with `VITE_API_BASE_URL=https://<API_DOMAIN>/api/v1`.

## Devices

ESP32 firmware connects to `tcp://<host>:${MQTT_PORT}` with `MQTT_USERNAME` / `MQTT_PASSWORD` and publishes to:

- `homelab/iot/env/{device_id}/sensor`
- `homelab/iot/devices/{device_id}/telemetry`
- `homelab/iot/devices/{device_id}/status` (retained, `online` / `offline`)

Payload formats: `iotlab-service/README.md`. Port 1883 is plain MQTT without TLS — keep it on the LAN or behind a
firewall.

Quick test from the host (needs `mosquitto-clients`):

```sh
mosquitto_pub -h localhost -u "$MQTT_USERNAME" -P "$MQTT_PASSWORD" \
  -t homelab/iot/env/test-01/sensor -m "{\"temp\":25.1,\"hum\":60,\"time\":$(date +%s000)}"
curl -k https://<API_DOMAIN>/api/v1/devices
```

## Database access from your PC

Postgres listens on `127.0.0.1` of the host only. Tunnel it:

```sh
ssh -N -L 15432:localhost:5432 user@vps-ip
psql -h localhost -p 15432 -U devlab telemetry
```

## Operations

```sh
docker compose up -d --build iotlab-service    # redeploy after code changes
docker compose logs -f traefik                 # certificate / routing problems
docker compose down                            # stop (data volumes are kept)
docker compose down -v                         # stop and DELETE all data (DB, broker, certs)
```

Volumes: `iotlab_tsdb-data` (database), `iotlab_mosquitto-data` (retained messages), `iotlab_traefik-acme` (certificates).

`iotlab-service/docker-compose.yml` is an older infra-only file (no app service, mounts `db/init.sql`); use the one in
this folder for deployment.
