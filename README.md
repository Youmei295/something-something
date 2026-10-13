# something-something — "Inbox"

A small self-hosted messaging server: visitors send a message (and optionally a
file) to the host, and the host replies. The backend is a set of Go
**microservices**; the frontend is a **static HTML/CSS/JS** site that talks only
to the API gateway.

## Architecture

```
Static frontend ──HTTP/JSON──▶ API gateway (:8080)
 (HTML/CSS/JS)                   │
                                 ├─▶ identity      (auth)
                                 ├─▶ submission    (visitor messages)
                                 ├─▶ conversation  (host inbox)
                                 ├─▶ attachment    (files, presigned URLs)
                                 └─▶ worker        (notify, reply, scan)
                                          │
                          Postgres  ◀─────┴─────▶  Object storage (S3 API)
```

- **Decoupled:** the frontend has no build step and no server-side code; it only
  knows an API base URL. The backend serves JSON only.
- **Microservices:** each context is its own binary (`cmd/identity`,
  `cmd/submission`, `cmd/conversation`, `cmd/attachment`), fronted by
  `cmd/gateway`. They can also all run in one process (`cmd/api`) for simplicity.
- The frontend and backend are independent: locally the frontend is served by a
  static file server, the backend by Go processes.

## Project layout

```
src/
├─ backend/      # Go module (services, gateway, worker, migrations)
└─ frontend/     # static site: *.html, style.css, api.js
deploy/
├─ compose/      # docker-compose for infra + backend
└─ caddy/        # Caddyfile(s) — TLS + API reverse proxy
```

## Prerequisites

- Go 1.26+
- Make
- Python 3 (only to serve the static frontend locally)
- Docker + Compose (for Postgres and object storage)

## Run locally

```bash
# 1. Configuration (once)
cp deploy/compose/.env.example deploy/compose/.env
cp src/backend/.env.example src/backend/.env

# 2. Infrastructure: Postgres + S3-compatible storage
make infra-up

# 3. Migrations + example host account
make seed

# 4. Backend (one terminal each; keep them running)
make api-run       # :8080  (all contexts in one process)
make worker-run    # background jobs (scan attachments, send email)

# 5. Frontend (one more terminal)
make web-serve     # http://localhost:3000
```

Open **http://localhost:3000**.

- Visitor: submit a message from the home page.
- Host: sign in at `login.html` with the example account below.

`make help` lists every target.

### Run as separate microservices (optional)

Instead of `make api-run`, start each service and the gateway:

```bash
make identity-run       # :8081
make submission-run     # :8082
make conversation-run   # :8083
make attachment-run     # :8084
make gateway-run        # :8080  ← point the frontend here
make worker-run
```

The frontend only ever talks to `:8080` (the gateway), so nothing else changes.

## Example host account

Seeded by `make seed` (idempotent; never overwrites an existing account):

| Email | Password |
|---|---|
| `host@example.com` | `change-me-too` |

Change `HOST_EMAIL` / `HOST_PASSWORD` in `.env` before seeding to use different
credentials.

## Configuration

Loaded from `src/backend/.env` (local) or `deploy/compose/.env` (Docker). Key values:

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `STORAGE_ENDPOINT` / `STORAGE_ACCESS_KEY` / `STORAGE_SECRET_KEY` | S3-compatible storage |
| `STORAGE_PUBLIC_ENDPOINT` | browser-reachable storage host for presigned URLs |
| `MAILER_DRIVER` | `console` (logs), `smtp`, or `postmark` |
| `HOST_EMAIL` / `HOST_PASSWORD` | example host account |
| `PUBLIC_BASE_URL` | frontend origin, used in email links |
| `ALLOWED_ORIGINS` | CORS allow-list (frontend origin) |
| `HTTP_ADDR` / `GATEWAY_ADDR` | listen addresses |

The frontend's API base is chosen in `src/frontend/api.js`: on `:3000` it uses
`http://localhost:8080`, otherwise `window.API_BASE` / `PROD_API_BASE` (set this
when the frontend is hosted elsewhere).

## API

All endpoints are JSON under `/api/v1`. Host routes require the session cookie
(the host signs in first). Full details live in the handler code
(`src/backend/internal/transport/http`).

| Method | Path | Auth | Purpose |
|---|---|---|---|
| POST | `/api/v1/auth/login` | public | host sign in (sets cookie) |
| POST | `/api/v1/auth/logout` | host | sign out |
| GET | `/api/v1/auth/me` | host | current host |
| POST | `/api/v1/messages` | public | visitor submits a message |
| GET | `/api/v1/visitor/threads/{token}` | public | visitor reads their thread |
| GET | `/api/v1/conversations` | host | list (status/search/paging) |
| GET | `/api/v1/conversations/{id}` | host | thread detail |
| POST | `/api/v1/conversations/{id}/replies` | host | reply |
| PATCH | `/api/v1/conversations/{id}` | host | mark read / archive |
| DELETE | `/api/v1/conversations/{id}` | host | delete |
| POST | `/api/v1/attachments/upload-url` | public | presigned upload URL |
| POST | `/api/v1/attachments/{id}/complete` | public | finalize an upload |
| GET | `/api/v1/attachments/{id}/download` | host | presigned download URL |
| GET | `/healthz`, `/readyz` | public | health / readiness |

Errors use `{ "error": "...", "fields": { ... } }`.

## Notes

- **Object storage:** the compose stack runs **SeaweedFS** (MinIO's community
  images were discontinued). It speaks the same S3 API; point `STORAGE_ENDPOINT`
  at any S3-compatible service to switch.
- **Attachments:** uploads go straight to storage via a presigned URL; downloads
  are served until the worker marks them `clean`. The API starts even if storage
  is down — uploads then return `503`.
- **CORS/cookies:** locally the frontend (`:3000`) and API (`:8080`) are the same
  site, so the session cookie works. Hosting the frontend on a different origin
  (e.g. GitHub Pages) needs extra setup (see below).

## Deploying the frontend to GitHub Pages

The workflow `.github/workflows/pages.yml` publishes `src/frontend/` to GitHub
Pages on every push to `main` that changes the frontend (or manually via
**Actions → Deploy frontend to GitHub Pages → Run workflow**). No backend is
involved.

One-time setup:

1. Repo **Settings → Pages → Build and deployment → Source: GitHub Actions**
   (on the free plan the repo must be public).
2. Merge to `main` (or run the workflow). The site appears at
   `https://<user>.github.io/<repo>/`.

Notes:
- The site is served under the `/<repo>/` subpath; all paths in `src/frontend`
  are relative, so this works as-is.
- `src/frontend/.nojekyll` disables Jekyll processing.
- Until the backend is reachable, the UI loads but API calls fail. When you have
  a backend origin, set `PROD_API_BASE` in `src/frontend/api.js` (e.g.
  `"https://api.example.com"`) and add that origin to the backend's
  `ALLOWED_ORIGINS`. Cross-site hosting also needs a bearer-token auth option.

### Backend (later)

Run the compose stack on a host with a domain; Caddy terminates TLS and proxies
`/api/*` to the gateway (`SITE_ADDRESS=api.example.com`).
