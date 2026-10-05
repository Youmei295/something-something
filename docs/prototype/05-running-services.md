# Running the Services Separately

The backend can run in two modes that share the exact same code:

| Mode | Entry point | Processes | Use for |
|---|---|---|---|
| Modular monolith | `cmd/api` | 1 | Simplicity, local dev, small deploys |
| Microservices | `cmd/identity`, `cmd/submission`, `cmd/conversation`, `cmd/attachment`, `cmd/gateway` | 5 (+ `cmd/worker`) | Mimicking the target architecture, independent scaling/deploys |

Both modes use the same `internal` packages. A "service" is defined only by which
contexts it enables (`transporthttp.Enable`), so splitting or merging is a wiring
change, not a rewrite.

## Services and routes

| Service | Owns | Public routes | Protected routes |
|---|---|---|---|
| `identity` (`:8081`) | host auth | `POST /api/v1/auth/login` | `/auth/logout`, `/auth/me` |
| `submission` (`:8082`) | visitor intake | `POST /api/v1/messages`, `GET /api/v1/visitor/threads/{token}` | — |
| `conversation` (`:8083`) | host inbox | — | `/conversations*` |
| `attachment` (`:8084`) | files | `POST /attachments/upload-url`, `POST /attachments/{id}/complete` | `GET /attachments/{id}/download` |
| `gateway` (`:8080`) | routing | fronts all of the above | — |
| `worker` | async jobs | — | — |

`/healthz` and `/readyz` exist on every HTTP service.

## Run them locally

Start only the infrastructure (Postgres + MinIO), apply migrations, then run each
service in its own terminal:

```bash
# 1. Infrastructure
npm run infra:up        # Postgres + MinIO via docker

# 2. Migrations + host seed
npm run seed

# 3. Each service (separate terminals)
npm run identity        # :8081
npm run submission      # :8082
npm run conversation    # :8083
npm run attachment      # :8084
npm run gateway         # :8080

# 4. Worker
npm run worker

# 5. Frontend
npm run web             # http://localhost:3000
```

The npm scripts just wrap Go (`cd src/backend && HTTP_ADDR=:8081 go run ./cmd/identity`),
so you can also run them directly, e.g. `cd src/backend && go run ./cmd/identity`.
Build standalone binaries with `npm run backend:build` (output in `bin/`).

Point the frontend at the gateway (`src/frontend/.env.local`):
`NEXT_PUBLIC_API_BASE_URL=http://localhost:8080`.

## Run them with Docker

```bash
make micro-up    # deploy/compose/docker-compose.micro.yml
```

Each service is its own container; Caddy → `gateway` → services. Stop with
`make micro-down`.

## Dependencies by service

The composition root (`internal/bootstrap`) builds dependencies lazily, so each
process only pays for what it uses:

```
identity      → Postgres
submission    → Postgres (+ conversation repo for the visitor thread view)
conversation  → Postgres
attachment    → Postgres + MinIO/S3
worker        → Postgres + mailer
gateway       → upstream URLs only (no database)
```

## Running a subset

The gateway treats a blank upstream URL as "disabled", so you can develop one
service without starting the others:

```bash
GATEWAY_IDENTITY_URL=http://localhost:8081 \
GATEWAY_SUBMISSION_URL= \
GATEWAY_CONVERSATION_URL= \
GATEWAY_ATTACHMENT_URL= \
  make gateway-run
```

Requests to a disabled prefix return `404`; a configured but unreachable upstream
returns `502`.

## Notes and limitations

- **Shared database.** All services currently point at one Postgres and validate
  host sessions by reading the `sessions` table. This is intentional for the
  prototype; true database-per-service isolation is a later milestone
  (`docs/04-roadmap-v1.md`).
- **Shared HTTP contract.** All services share the `internal/transport/http`
  package with per-context route groups, so they cannot drift apart.
- **No service discovery.** The gateway uses static upstream URLs from config;
  this is fine for a fixed deployment and is the place a registry would plug in.
- **Extracting a service further** (separate module/repo, gRPC, own DB) is
  mechanical because services communicate only through the gateway, the shared
  DB, and the job table.
