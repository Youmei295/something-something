# something-something — "Inbox"

A self-hosted inbox server: visitors send messages and files to the host, and the
host replies. Built as a small, decoupled system for practicing real server
operations. See [`docs/`](./docs) for the proposal, architecture, and roadmap.

**Prototype v0.1** — the first working vertical slice:

- Public submission page (message + one 5 MB attachment).
- Host login, inbox list, conversation view, threaded replies.
- Outbound email for host notifications and visitor replies.
- Postgres, MinIO (S3-compatible) storage, and a background worker.

## Architecture

```
Browser ──▶ Caddy ──┬──▶ Next.js (web)
                    └──▶ Go API ──┬──▶ Postgres      (data)
                                  ├──▶ MinIO         (attachments)
                                  └──▶ Mailer        (provider/SMTP/console)
                                        ▲
                                   Go worker (jobs: notify, reply, scan, cleanup)
```

The backend follows a ports-and-adapters layout so the domain and use cases stay
independent of HTTP, SQL, and vendors:

```
src/
├─ backend/                     # Go module
│  ├─ cmd/{api,worker,migrate,identity,submission,conversation,attachment,gateway}
│  └─ internal/
│     ├─ domain/                # entities + sentinel errors (no deps)
│     ├─ ports/                 # interfaces the core depends on
│     ├─ app/                   # use cases (services + worker)
│     ├─ adapters/{postgres,storage,mailer,auth}
│     ├─ transport/{http,gateway}
│     ├─ platform/{clock,id,link,logger,httpserver}
│     └─ bootstrap/             # composition root (one place that wires deps)
└─ frontend/                    # Next.js app
```

Adding a feature usually means: a domain type, a port method, an adapter
implementation, a use case, and a handler — each independently testable. The
worker and API share the same composition root, so swapping the Postgres job
table for an event bus (v0.2) is a localized change.

## Quick start

### With Docker (recommended)

```bash
cp deploy/compose/.env.example deploy/compose/.env
# edit secrets, then:
make up          # docker compose up -d --build
```

Open http://localhost and sign in at `/login` using `HOST_EMAIL` / `HOST_PASSWORD`.

### Local development

Backend services run with Go, the frontend with npm. Scripts at the repo root
wrap both so you never have to remember paths:

```bash
cp deploy/compose/.env.example deploy/compose/.env   # infra (once)
cp src/backend/.env.example src/backend/.env         # backend (once)

npm run infra:up          # start Postgres + MinIO (docker)
npm run seed              # migrate + create the host account

npm run api               # all-in-one API on :8080   (terminal 1)
npm run worker            # background jobs            (terminal 2)

npm run web:install       # first time only
npm run web               # http://localhost:3000      (terminal 3)
```

Full walkthrough, configuration, and troubleshooting:
[`docs/STARTUP.md`](./docs/STARTUP.md).

You can build binaries too and run them directly:

```bash
npm run backend:build     # -> bin/{api,worker,migrate,identity,...}
./bin/api
```

## Common commands

```bash
npm run backend:test      # go test ./...
npm run backend:lint      # gofmt + go vet
npm run web:build         # next build

make help                 # equivalent targets, if you prefer make
make check                # lint + tests + frontend build
```

## Run as microservices

The same code can run as one process (`cmd/api`) or as independent services wired
by the API gateway. A service is defined only by which contexts it enables, so
splitting/merging is configuration, not a rewrite.

```bash
# Docker
make micro-up            # Caddy → gateway → identity/submission/conversation/attachment + worker

# Local processes (each in its own terminal)
npm run identity         # :8081
npm run submission       # :8082
npm run conversation     # :8083
npm run attachment       # :8084
npm run gateway          # :8080   ← point the frontend here
npm run worker           # background jobs
```

Details, ports, route ownership, and the dependency graph are in
[`docs/prototype/05-running-services.md`](./docs/prototype/05-running-services.md).

## Configuration

All configuration is environment-based; see
[`deploy/compose/.env.example`](./deploy/compose/.env.example) for the full list.
Key values: `DATABASE_URL`, `STORAGE_*`, `MAILER_DRIVER`, `HOST_*`, and
`PUBLIC_BASE_URL`.

## Deployment

See [`deploy/README.md`](./deploy/README.md) for the runbook (TLS, storage
subdomain, email provider, backups, rollback) and
[`.github/workflows/`](./.github/workflows) for CI and deploy automation.

## Roadmap

v0.1 (this) → v0.2 hardening (NATS, ClamAV, Redis, observability) → v0.3 real
email (SMTP intake) → v0.4 multi-host → v1.0 stable. Details in
[`docs/04-roadmap-v1.md`](./docs/04-roadmap-v1.md).
