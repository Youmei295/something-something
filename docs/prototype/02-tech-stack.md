# Prototype Tech Stack — Decisions (ADR)

These decisions are **binding for later versions** unless explicitly superseded by a
new ADR. They were chosen to minimize operational burden for a single developer while
keeping a clean path to the microservice architecture.

Format: **Decision · Status · Context · Rationale · Consequences · Revisit when.**

---

## ADR-001 — Backend language: Go

- **Status:** Accepted
- **Context:** Need to run several small services on one small VPS, parse MIME later,
  and handle concurrent requests with low memory.
- **Decision:** Go for the API, worker, and all future backend services.
- **Rationale:** Single static binaries, tiny containers, low RAM, strong stdlib for
  HTTP/JSON/binary parsing, excellent concurrency. Easy to operate solo.
- **Consequences:** More verbose than Python/TS; team must be comfortable with Go.
  Shared code lives in an internal module (`packages/go`).
- **Revisit when:** never expected; only if a required library is missing.

## ADR-002 — Backend HTTP framework: chi (net/http)

- **Status:** Accepted
- **Context:** Framework choice for Go services.
- **Decision:** Start with **chi** (idiomatic `net/http` middleware stack).
- **Rationale:** Stdlib-compatible, minimal, easy to test, no lock-in.
- **Consequences:** Some features (gRPC, codegen) added manually later.
- **Revisit when:** moving service-to-service calls to gRPC (v0.2).

## ADR-003 — Frontend: Next.js (React) + TypeScript

- **Status:** Accepted
- **Context:** Need a public SSR page (SEO/fast first paint) and an authenticated
  dashboard.
- **Decision:** **Next.js App Router + TypeScript**, styled with **Tailwind CSS** and
  **shadcn/ui**; server state via **TanStack Query**.
- **Rationale:** One framework for both surfaces, strong ecosystem, easy deploy as a
  container, SSR for the public page.
- **Consequences:** Node runtime in the deploy; keep the image slim.
- **Revisit when:** never expected.

## ADR-004 — Database: PostgreSQL (single instance, one schema per service)

- **Status:** Accepted
- **Context:** Relational data (conversations, messages, users) and future full-text.
- **Decision:** **PostgreSQL 16**. In the prototype one database; as services split,
  one schema/role per service, still on one instance until scale requires more.
- **Rationale:** Reliable, JSONB, full-text search, familiar, easy backups.
- **Consequences:** Must enforce schema ownership by convention until it is physical.
- **Revisit when:** any single service's load justifies a dedicated instance (v1.0).

## ADR-005 — Data access: sqlc (typed SQL)

- **Status:** Accepted
- **Context:** Need compile-time-safe queries without a heavy ORM.
- **Decision:** **sqlc** for Go query generation + **goose** for migrations.
- **Rationale:** SQL-first, type-safe, zero runtime reflection, easy to review.
- **Consequences:** Migrations are explicit; no automatic relationship loading.
- **Revisit when:** never expected.

## ADR-006 — Object storage: MinIO (S3-compatible)

- **Status:** Accepted
- **Context:** Store attachments without exposing the filesystem; must migrate to S3
  later without code changes.
- **Decision:** **MinIO** in the prototype and local dev; **AWS S3** in production
  once scaling. Use **presigned URLs** for upload/download.
- **Rationale:** S3 API compatibility means no application changes to move.
- **Consequences:** One more container; needs persistent volume + credentials.
- **Revisit when:** managed S3 becomes cheaper/simpler (v1.0).

## ADR-007 — Async jobs: Postgres job table (no message bus yet)

- **Status:** Accepted (temporary)
- **Context:** Prototype needs to send emails and retry without a message broker.
- **Decision:** Use a **Postgres-backed job table** polled by the Go worker with
  `FOR UPDATE SKIP LOCKED`.
- **Rationale:** Zero extra infrastructure; correct enough for prototype volume.
- **Consequences:** Not suitable for high throughput or multiple event consumers.
- **Revisit when:** v0.2 — introduce **NATS JetStream** and migrate jobs to events.

## ADR-008 — Outbound email: transactional provider

- **Status:** Accepted
- **Context:** Must reliably deliver host replies and notifications; running an MTA
  requires IP warmup and reputation management.
- **Decision:** Send via a **transactional provider** (Postmark preferred; AWS SES as
  the cheaper fallback), abstracted behind a `Mailer` interface.
- **Rationale:** Deliverability, bounce/complaint webhooks, DKIM handled for us.
- **Consequences:** Vendor cost/dependency; abstraction makes replacement easy.
- **Revisit when:** running our own MTA becomes the explicit learning goal (far future).

## ADR-009 — Authentication: session cookie for host, opaque token for visitor

- **Status:** Accepted
- **Context:** Single host account; visitors have no account.
- **Decision:** Host auth via **httpOnly session cookie** backed by server-side
  sessions stored in Postgres; passwords hashed with **argon2id**. Visitors are
  identified by an **opaque thread token** embedded in reply links.
- **Rationale:** Simpler and safer than JWT in the browser for a single-host app;
  opaque tokens avoid leaking visitor data.
- **Consequences:** Sessions require DB lookups; add Redis cache in v0.2.
- **Revisit when:** multiple hosts / OIDC / MFA (v0.3+).

## ADR-010 — Deployment: Docker Compose on a single VPS behind Caddy

- **Status:** Accepted
- **Context:** Solo operator, small budget, wants real-world ops practice.
- **Decision:** **Docker Compose** on one VPS; **Caddy** as reverse proxy with
  automatic HTTPS. Containers: `web`, `api`, `worker`, `postgres`, `minio`, `caddy`.
- **Rationale:** Minimal moving parts, automatic TLS, trivially reproducible.
- **Consequences:** Single point of failure; no zero-downtime deploys yet.
- **Revisit when:** v1.0 — **Kubernetes (k3s)** or Nomad for autoscaling/HA.

## ADR-011 — CI/CD: GitHub Actions

- **Status:** Accepted
- **Context:** Need build, test, image publish, and deploy on push.
- **Decision:** **GitHub Actions** builds and tests on PR; on `main` it builds images,
  pushes to GHCR, and deploys via SSH + `docker compose pull/up`.
- **Rationale:** Free for public/private repos within limits; ubiquitous.
- **Consequences:** Deploy step needs an SSH key secret and a `docker compose` file
  on the host.
- **Revisit when:** adopting Kubernetes (switch to GitOps/Argo CD).

## ADR-012 — Observability: structured logs + health endpoints (prototype)

- **Status:** Accepted
- **Context:** Need to debug a live server without building a monitoring stack yet.
- **Decision:** JSON structured logs with a request ID; `/healthz` and `/readyz` on
  every service. Full metrics/tracing deferred.
- **Rationale:** Cheapest useful signal; keeps prototype deployable on a small VPS.
- **Consequences:** No dashboards/alerts until v0.2.
- **Revisit when:** v0.2 — add **Prometheus/Grafana + Loki + OpenTelemetry**.

## ADR-013 — Repo layout: monorepo

- **Status:** Accepted
- **Context:** Single developer, shared contracts and types across services.
- **Decision:** One **monorepo** with `src/frontend`, `src/backend`, `packages/`, `deploy/`,
  `docs/`.
- **Rationale:** Atomic changes across the contract boundary, one CI pipeline.
- **Consequences:** CI must be selective (only build changed components).
- **Revisit when:** team grows and release cadence diverges (v1.0+).

---

## Frozen Summary

| Concern | Prototype choice | Later target |
|---|---|---|
| Backend | Go + chi | Go + gRPC between services |
| Frontend | Next.js + TS + Tailwind | same |
| Database | Postgres + sqlc/goose | Postgres per service |
| Storage | MinIO (S3 API) | AWS S3 |
| Async | Postgres job table | NATS JetStream |
| Email out | Postmark/SES | same |
| Auth | session cookie + argon2id | + Redis, MFA, roles |
| Deploy | Compose + Caddy, 1 VPS | k3s + cert-manager |
| CI/CD | GitHub Actions | GitOps/Argo CD |
| Observability | JSON logs + health | Prometheus/Grafana/Loki/OTel |

**Rule of change:** any deviation from a `Frozen Summary` value requires a new ADR in
this file, dated, with the reason and migration note.
