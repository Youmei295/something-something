# Deployment & Runbook

The prototype runs entirely with Docker Compose: **Caddy → web + api**, backed by
**Postgres**, **MinIO**, and a **worker**.

There are two compose files. `docker-compose.yml` runs the API as a single
process; `docker-compose.micro.yml` runs each service separately behind the Go
gateway (`make micro-up`). See
[`docs/prototype/05-running-services.md`](../docs/prototype/05-running-services.md).

## Quick start (local)

```bash
cp deploy/compose/.env.example deploy/compose/.env
# edit secrets in .env
docker compose -f deploy/compose/docker-compose.yml --env-file deploy/compose/.env up -d --build
```

Then open http://localhost (Caddy). Sign in at `/login` with `HOST_EMAIL` /
`HOST_PASSWORD`. The host account is seeded by the `migrate` service on first run.

MinIO console: http://localhost:9001 (credentials from `.env`).

## Services

| Service | Image / build | Role |
|---|---|---|
| `caddy` | `caddy:2-alpine` | TLS + reverse proxy (`/api/*` → api, else → web). |
| `web` | `src/frontend` | Next.js frontend. |
| `api` | `src/backend` | Public + host HTTP API. |
| `worker` | `src/backend` | Sends email and processes jobs. |
| `migrate` | `src/backend` | Runs migrations + seeds host, then exits. |
| `postgres` | `postgres:16-alpine` | Primary database. |
| `minio` | `minio/minio` | S3-compatible attachment storage. |

## Production checklist

1. Set `SITE_ADDRESS=inbox.example.com` and `PUBLIC_BASE_URL=https://inbox.example.com`.
   Caddy obtains TLS automatically.
2. Attachments need a browser-reachable storage host. Put MinIO behind a
   subdomain (e.g. `storage.example.com`) and set:
   - `STORAGE_PUBLIC_ENDPOINT=storage.example.com`
   - `STORAGE_USE_SSL=true`
   `STORAGE_ENDPOINT` stays `minio:9000` for server-side access.
3. Configure email: set `MAILER_DRIVER=postmark` and `POSTMARK_SERVER_TOKEN`, and
   verify the `MAIL_FROM_ADDRESS` domain with the provider.
4. Set `SESSION_COOKIE_SECURE=true`.
5. Use strong `POSTGRES_PASSWORD`, `MINIO_ROOT_PASSWORD`, `HOST_PASSWORD`.

## Common operations

```bash
# Logs
docker compose -f deploy/compose/docker-compose.yml --env-file deploy/compose/.env logs -f api worker

# Re-run migrations after a deploy
docker compose ... run --rm migrate

# Restart one service
docker compose ... restart api

# Update images and roll out
git pull
docker compose ... up -d --build
```

## Backups

```bash
# Postgres dump
docker compose ... exec postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" > backup-$(date +%F).sql

# MinIO data lives in the `miniodata` volume; snapshot it or use `mc mirror`.
```

A restore drill is part of the v0.2 hardening milestone.

## Rollback

Images are rebuilt from the tagged commit. To roll back, check out the previous
tag and redeploy:

```bash
git checkout v0.1-M4
docker compose ... up -d --build
```

## Troubleshooting

- **`/readyz` returns 503**: the API cannot reach Postgres. Check `DATABASE_URL`
  and that `postgres` is healthy.
- **Uploads fail in the browser**: `STORAGE_PUBLIC_ENDPOINT` must be reachable
  from the browser and its scheme must match `STORAGE_USE_SSL`.
- **Attachment download blocked**: the scan job may not have run. Check the
  `worker` logs; attachments are only downloadable once `scan_status = clean`.
- **No host notification emails**: with `MAILER_DRIVER=console` emails are logged,
  not sent. Configure a real provider for delivery.
