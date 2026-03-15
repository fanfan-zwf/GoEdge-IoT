# Copilot / AI Agent Instructions for GoEdge-IoT

Purpose: Give AI coding agents the minimal, actionable context to be productive in this repository.

Big picture
- Monorepo with two Go backend services and two Vue3 frontends:
  - `Iot-User-Service` — Go microservice for user/auth (entry: `Iot-User-Service/main.go`, config: `Iot-User-Service/config.yaml`).
  - `Iot-Data-Service` — Go microservice for data ingestion/point storage (check `Iot-Data-Service/` for `main.go`).
  - `Iot-User-Web` and `Iot-Data-Web` — Vite + Vue3 frontends (use `package.json` scripts).
- Cross-component integrations: MySQL, Redis, InfluxDB, WebSocket/real-time push (see repository README and service `db/` folders).

Developer workflows (how to run & build)
- Go services (per-service):
  - Run for development: `cd Iot-User-Service && go run main.go` (same pattern for `Iot-Data-Service`).
  - Build: `cd Iot-User-Service && go build -o bin/iot-user-service .`
  - Config is loaded from `config.yaml` in each service root — check `API.post` and DB sections for ports/credentials.
- Frontends:
  - Install deps (repo uses pnpm lockfiles): `cd Iot-User-Web && pnpm install` (or `npm install`/`yarn` if preferred).
  - Start dev server: `pnpm dev` (script `dev` in `package.json`).
  - Build: `pnpm build` (runs `vite build`).

Project-specific conventions & patterns
- Go layout: packages organized under `db/`, `web/`, `Init/`, `cloud/`. Services import local packages as `main/...` (see `Iot-User-Service/main.go`).
- Configuration: YAML files at service roots (`config.yaml`). Avoid hardcoding credentials — copy and edit `config.yaml` for local testing.
- Web apps: use Vite + Vue3, `pinia` for state; tests use `vitest` and `playwright` for e2e (see `package.json`).

Integration and data flow notes
- User service exposes HTTP API (port configured in `config.yaml`, example: 8101 for `Iot-User-Service`).
- Data service writes time-series data into InfluxDB (`Iot-Data-Service/db/influxdb`) and also uses MySQL/Redis for metadata and caching.
- Real-time updates use WebSocket layers in `web/` folders (see `Iot-Data-Service/web` and `Iot-User-Service/web`).

Guidance for AI agents (what to do first)
- Search for `main.go` under service folders to find entry points.
- Inspect `config.yaml` before running a service — it contains ports and DB connection strings.
- When changing APIs, update corresponding frontend calls in `Iot-User-Web/src/utils/api.ts` or similar utility files.
- Avoid modifying database schemas without a migration plan — check `db/mysql` folders for schema usage.

Examples (concrete patterns to follow)
- Adding an HTTP handler: add method in `web/` package and call `go web.Web()` from `main.go` (see `Iot-User-Service/main.go`).
- Database access: use the `db/mysql` package (see `Iot-User-Service/db/mysql/mysql.go`) and close connections in `exit()` in `main.go`.

If you need more context
- Read the top-level README.md for product intent and key technologies.
- If unsure about run steps for a service, open that service's `README.md` or `config.yaml`.

— End of agent instructions —

Please review and tell me any missing details or preferred conventions to include.
