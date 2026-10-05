# Startup Guide

How to get **Inbox** running from a fresh clone. There are three ways to start;
pick the one that fits what you are doing.

| Path | Best for | Time |
|---|---|---|
| [A. Docker (monolith)](#a-docker-monolith) | Just try it end to end | ~2 min |
| [B. Docker (microservices)](#b-docker-microservices) | Mimicking the target architecture | ~3 min |
| [C. Local processes](#c-local-processes) | Active development with hot reload | ~5 min |

Related: [`prototype/05-running-services.md`](./prototype/05-running-services.md)
(service ports and routing) and [`../deploy/README.md`](../deploy/README.md)
(production runbook).

---

## Prerequisites

| Tool | Version | Needed for |
|---|---|---|
| Docker + Compose | Compose v2+ | A, B, and infra in C |
| Node.js | 22+ | Frontend (all paths) |
| Go | 1.26+ | Backend (path C, and building binaries) |
| Make | optional | `make` shortcuts |

Check:

```bash
docker --version && docker compose version
node --version
go version
```

---

## First-time setup (all paths)

The repo runs on environment files. Copy the examples once:

```bash
cp deploy/compose/.env.example deploy/compose/.env     # infra + Docker stack
cp src/backend/.env.example src/backend/.env           # local `go run` backend (path C)
```

Edit the passwords in `deploy/compose/.env` if you like, but keep
`POSTGRES_PASSWORD` / `MINIO_ROOT_PASSWORD` in both files consistent.

Key values you will use to sign in:

- `HOST_EMAIL` (default `host@example.com`)
- `HOST_PASSWORD` (default `change-me-too`)

---

## A. Docker (monolith)

Everything (web, api, worker, Postgres, MinIO, Caddy) in one stack.

```bash
npm run up        # docker compose up -d --build   (or: make up)
```

Wait for the `migrate` service to finish (it runs migrations and seeds the host
account), then open:

- App: **http://localhost**
- MinIO console: **http://localhost:9001**

Sign in at **http://localhost/login**.

```bash
npm run logs      # follow logs
npm run down      # stop
```

---

## B. Docker (microservices)

Each bounded context runs as its own container behind the Go gateway.

```bash
npm run micro:up  # docker compose -f ...micro.yml up -d --build
npm run micro:down
```

Topology: `Caddy → gateway → identity / submission / conversation / attachment`
plus `worker`. Same URLs as path A.

---

## C. Local processes

Infrastructure in Docker, application processes locally with `go run` / npm —
best for editing code.

### 1. Start infrastructure

```bash
npm run infra:up          # Postgres + MinIO only
```

### 2. Migrate and seed

```bash
npm run seed              # go run ./cmd/migrate -seed  (reads src/backend/.env)
```

### 3. Start the backend

Either the all-in-one API:

```bash
npm run api               # :8080
npm run worker            # background jobs (second terminal)
```

…or the individual services (one terminal each), plus the gateway:

```bash
npm run identity          # :8081
npm run submission        # :8082
npm run conversation      # :8083
npm run attachment        # :8084
npm run gateway            # :8080  ← point the frontend here
npm run worker
```

Each script is just `cd src/backend && HTTP_ADDR=:PORT go run ./cmd/<service>`,
so you can also run them directly. Build standalone binaries with
`npm run backend:build` (output in `bin/`).

### 4. Start the frontend

```bash
cp src/frontend/.env.example src/frontend/.env.local   # once
npm run web:install                                     # once
npm run web                # http://localhost:3000
```

`src/frontend/.env.local` controls the API origin:

- Path C with the gateway: `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080`
- Path C with only `npm run api`: `NEXT_PUBLIC_API_BASE_URL=http://localhost:8080`
  works too (the default `api` listens on `:8080`).

---

## Verify it works

1. Open the app (http://localhost or http://localhost:3000).
2. On the home page, send a message (optionally attach an image/PDF ≤ 5 MB).
3. You get a confirmation with a link to the conversation.
4. Sign in as host at `/login` with `HOST_EMAIL` / `HOST_PASSWORD`.
5. Open the message in `/inbox`, reply, and (with a real mailer) the visitor
   receives the reply email.

Health checks:

```bash
curl -fsS http://localhost/healthz      # via Caddy (paths A/B)
curl -fsS http://localhost:8080/healthz # gateway (path C)
curl -fsS http://localhost:8080/readyz  # checks DB / upstreams
```

---

## Service map

| Service | Local port | Owns |
|---|---|---|
| `gateway` | 8080 | routes `/api/v1/*` to the services |
| `identity` | 8081 | `/auth/login`, `/auth/logout`, `/auth/me` |
| `submission` | 8082 | `POST /messages`, visitor thread view |
| `conversation` | 8083 | `/conversations/*` (list, read, reply, archive) |
| `attachment` | 8084 | upload URL, complete, download |
| `worker` | — | notifications, outbound mail, scan, cleanup |
| `web` | 3000 | Next.js frontend |
| Postgres | 5432 | database |
| MinIO | 9000 / 9001 | attachments / console |

---

## Configuration

Backend config is environment-based and loaded from `src/backend/.env` for local
runs (`ENV_FILE` overrides the path; real env vars always win). The most
important variables:

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `STORAGE_*` | MinIO/S3 endpoint, keys, bucket, public endpoint |
| `MAILER_DRIVER` | `console` (logs), `smtp`, or `postmark` |
| `POSTMARK_SERVER_TOKEN` | required when `MAILER_DRIVER=postmark` |
| `HOST_EMAIL` / `HOST_PASSWORD` | seeded host account |
| `PUBLIC_BASE_URL` / `ALLOWED_ORIGINS` | links in email + CORS |
| `HTTP_ADDR` / `GATEWAY_ADDR` | listen addresses |
| `GATEWAY_*_URL` | gateway upstreams (blank disables a route) |

Full list: [`deploy/compose/.env.example`](../deploy/compose/.env.example) and
[`src/backend/.env.example`](../src/backend/.env.example).

---

## Common commands

```bash
# Stack
npm run up / down / logs          # monolith
npm run micro:up / micro:down     # microservices
npm run infra:up / infra:down     # just Postgres + MinIO

# Backend
npm run migrate / seed
npm run api / identity / submission / conversation / attachment / gateway / worker
npm run backend:build             # binaries in bin/
npm run backend:test              # go test ./...
npm run backend:lint              # gofmt + go vet

# Frontend
npm run web:install / web / web:build / web:lint
```

The same operations are available through `make` (`make help` lists them).

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `readyz` returns 503 | Postgres unreachable. Check `DATABASE_URL` and `npm run infra:up`. |
| Login fails with seeded user | Ensure `migrate -seed` ran and you copied `.env` before starting. |
| Frontend calls fail with CORS/401 | Set `NEXT_PUBLIC_API_BASE_URL` to the API/gateway origin and restart `npm run web`. |
| Upload fails in the browser | `STORAGE_PUBLIC_ENDPOINT` must be browser-reachable and match `STORAGE_USE_SSL`. |
| Download button disabled / 403 | The scan job has not run; ensure `npm run worker` is running (attachments become downloadable at `scan_status=clean`). |
| No emails arrive | `MAILER_DRIVER=console` only logs email. Configure `postmark` (or `smtp`) to send. |
| Port already in use | Change `HTTP_ADDR` / `GATEWAY_ADDR`, or free the port. |

Reset everything (removes database and object storage):

```bash
npm run down
docker volume rm inbox_pgdata inbox_miniodata
```

---

## Next steps

- Read the architecture: [`02-architecture.md`](./02-architecture.md).
- Understand the split: [`prototype/05-running-services.md`](./prototype/05-running-services.md).
- See what is planned: [`04-roadmap-v1.md`](./04-roadmap-v1.md).
