# Proposed Tech Stack

Two tracks are given: a **pragmatic default** (fast to build, easy to operate) and
**alternatives** if you want to practice a particular ecosystem. The default is
chosen to keep the microservice boundary real but the operational burden low.

## 1. Recommended Default

| Layer | Choice | Why |
|---|---|---|
| Frontend framework | **Next.js (React) + TypeScript** | SSR for the public page, app router for dashboard, huge ecosystem. |
| Frontend styling | **Tailwind CSS + shadcn/ui** | Fast, consistent UI without a heavy design system. |
| Client data | **TanStack Query** | Caching, retries, server-state handling. |
| API Gateway / BFF | **Traefik or Caddy + a Node/Go gateway**, or **Kong/APISIX** | Routing, TLS, rate limiting. |
| Backend language | **Go** | Small static binaries, great concurrency, low memory — ideal for many services on one box. |
| Backend framework | **Fiber / Echo / chi** | Lightweight HTTP, easy to containerize. |
| Alternative backend | **Node.js (NestJS)** or **Python (FastAPI)** | NestJS gives structure for many services; FastAPI is fast to prototype. |
| Internal RPC | **gRPC (+ Protobuf)** | Typed service-to-service contracts. REST/JSON for public API. |
| Message bus | **NATS JetStream** (or RabbitMQ) | Lightweight, at-least-once, easy on a VPS. Kafka only if you need long retention/streaming. |
| Primary database | **PostgreSQL** (one DB or schema per service) | Reliable, relational, JSONB, full-text. |
| Cache / rate limit | **Redis** | Rate limiting, sessions, idempotency keys. |
| Object storage | **MinIO** (self-host) → S3 when scaling | S3-compatible, presigned URLs. |
| Malware scanning | **ClamAV** | Open-source, container-friendly. |
| Outbound email | **Postmark / AWS SES / Mailgun** | Deliverability without running an MTA. |
| Inbound SMTP (P2) | **Custom Go SMTP service** (e.g. `emersion/go-smtp`) | Controlled parsing, feeds the same pipeline. |
| Auth | **JWT + refresh**, `bcrypt`/`argon2` for hashes | Standard, stateless. |
| Search (P2) | **Meilisearch** | Simple, fast full-text; OpenSearch if you need more. |
| Background jobs | **The message bus + workers** (or River/Asynq) | Retries, scheduling, DLQ. |
| Containerization | **Docker + Docker Compose** | Single-VPS deployment for P0–P1. |
| Orchestration (P2) | **Kubernetes** (k3s for cheap) or **Nomad** | Autoscaling, self-healing, rolling deploys. |
| Reverse proxy / TLS | **Caddy** (auto HTTPS) or **Traefik**; `cert-manager` on K8s | Zero-config certificates. |
| CI/CD | **GitHub Actions** | Build, test, scan, push images, deploy. |
| Observability | **Prometheus + Grafana**, **Loki**, **OpenTelemetry**, **Sentry** | Metrics, logs, traces, error tracking. |
| Secrets | **Docker secrets / SOPS+age** → **Vault** | Avoid plaintext env files. |
| Testing | **Go test + testcontainers**, **Playwright** (E2E), **Vitest** (frontend) | Unit → integration → E2E. |
| IaC (P2) | **Terraform + Ansible** | Reproducible infra and host config. |

## 2. Minimal P0 Stack (start here)

If the full table feels heavy, this is enough for the first working version:

- **Frontend**: Next.js + TypeScript + Tailwind.
- **Backend**: Go (single "core API" service) + PostgreSQL + Redis.
- **Files**: MinIO with presigned uploads (ClamAV added in P2).
- **Email**: provider HTTP API (SES/Postmark) for notifications and replies.
- **Deploy**: one VPS, Docker Compose, Caddy for automatic HTTPS.
- **CI**: GitHub Actions builds and deploys on push.

Add the message bus, gRPC split, SMTP intake, and Kubernetes only when the
corresponding requirement appears.

## 3. Alternative Stacks by Goal

| If you want to practice... | Choose |
|---|---|
| TypeScript everywhere | Next.js + NestJS microservices + Prisma + NATS + Postgres. |
| Python | Next.js + FastAPI services + SQLAlchemy + RabbitMQ + Celery. |
| .NET | Next.js/Blazor + ASP.NET Core services + EF Core + MassTransit + RabbitMQ. |
| Java | React + Spring Boot + Kafka + Postgres. |
| Rust | React + Axum services + SQLx + NATS. |
| Running a real MTA | Postfix/Dovecot + custom policy/milter service. |

## 4. Rationale Highlights

- **Go for backend**: many services on a small server benefit from low memory and
  simple single-binary deploys; strong standard library for HTTP/SMTP/binary parsing.
- **Postgres per service**: keeps clear ownership boundaries while staying single-engine
  (easy backups, one skill set).
- **NATS over Kafka**: the workload is request/event driven, not big-data streaming;
  JetStream gives persistence and retries without ZooKeeper/KRaft overhead.
- **MinIO → S3**: start self-hosted, migrate without code changes (S3 API compatible).
- **Caddy for solo ops**: automatic TLS removes a whole class of certificate pain.
- **Provider email**: running your own outbound MTA means IP warmup and reputation
  management; defer that until it is itself the learning goal.

## 5. Suggested Repo Layout (monorepo)

```
/
├─ src/
│  ├─ frontend/            # Next.js frontend
│  └─ backend/             # Go module
│     ├─ cmd/
│     │  ├─ api/           # all-in-one modular monolith
│     │  ├─ identity/      # standalone services
│     │  ├─ submission/
│     │  ├─ conversation/
│     │  ├─ attachment/
│     │  ├─ gateway/       # API gateway / BFF
│     │  ├─ worker/
│     │  └─ migrate/
│     └─ internal/         # domain, ports, app, adapters, transport
├─ packages/
│  ├─ proto/               # shared Protobuf / API contracts
│  ├─ events/              # event schemas (message.received, ...)
│  └─ ui/                  # shared frontend components
├─ deploy/
│  ├─ compose/             # docker-compose for local + VPS
│  └─ k8s/                 # manifests/helm (P2)
├─ docs/
└─ .github/workflows/
```

## 6. Decision Summary

Build **Next.js + Go + Postgres + Redis + MinIO + NATS**, deploy with **Docker
Compose on a single VPS behind Caddy**, send mail through a **transactional email
provider**, and grow into **gRPC + Kubernetes + SMTP intake** only when a concrete
need demands it.
