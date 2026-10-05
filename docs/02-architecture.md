# Architecture Plan

## 1. High-Level Overview

```
                    ┌─────────────────────────────┐
   Visitor ───────▶ │  Frontend (Next.js)          │
                    │  - public submit page        │
                    │  - host dashboard            │
                    └───────────────┬─────────────┘
                                    │ HTTPS (REST/JSON)
                                    ▼
                    ┌─────────────────────────────┐
                    │  API Gateway / BFF           │
                    │  routing, auth, rate limit,  │
                    │  request validation, CORS    │
                    └───┬───────┬───────┬──────┬───┘
                        │       │       │      │
       ┌────────────────┘       │       │      └────────────────┐
       ▼                        ▼       ▼                       ▼
┌─────────────┐        ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
│ Submission  │        │ Conversation │ │  Attachment  │ │  Identity    │
│ Service     │        │ /Thread Svc  │ │  Service     │ │  /Auth Svc   │
└──────┬──────┘        └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
       │                      │                │                │
       │        ┌─────────────┴─────┐          │                │
       │        ▼                   ▼          ▼                │
       │  ┌───────────┐     ┌──────────────┐ ┌──────────┐        │
       │  │ Postgres  │     │  Object      │ │ Postgres │        │
       │  │ (threads) │     │  Storage S3  │ │ (auth)   │        │
       │  └───────────┘     └──────────────┘ └──────────┘        │
       │                                                        │
       ▼                                                        │
┌─────────────────┐   events    ┌──────────────────┐           │
│  Message Bus    │◀───────────▶│ Notification Svc │           │
│ (NATS/RabbitMQ) │             │ email/webhook    │           │
└────────┬────────┘             └──────────────────┘           │
         │                                                      │
         ▼                                                      │
┌─────────────────┐   sends replies    ┌──────────────────┐    │
│  Outbound Mail  │◀───────────────────│  SMTP Intake Svc │    │
│  Service        │                    │  (optional)      │    │
└─────────────────┘                    └──────────────────┘    │
         ▲                                                      │
         └──────────────────────────────────────────────────────┘
                     Admin / Host auth
```

## 2. Services

### 2.1 API Gateway / BFF
- Single public entry point. Terminates TLS (via reverse proxy).
- Responsibilities: routing, JWT/session verification, rate limiting, request
  validation, CORS, API aggregation for the dashboard, idempotency keys.
- Stateless; scales horizontally.

### 2.2 Submission Service
- Public endpoint `POST /messages` (visitor-facing).
- Validates input, runs captcha verification, normalizes into the internal
  message model, creates a conversation via the Conversation Service, and requests
  attachment upload slots.
- Publishes `message.received` events to the bus.
- This is the only service exposed to unauthenticated visitors (behind gateway).

### 2.3 Conversation / Thread Service
- Owns conversations, messages, participants, labels, read state, search index.
- Endpoints for host: list threads, read, reply, archive, delete.
- Stores metadata in its own Postgres schema/DB (database-per-service).

### 2.4 Attachment Service
- Issues presigned upload URLs, stores blobs in S3/MinIO.
- Enforces size/type/count limits.
- Runs malware scanning (ClamAV sidecar or async worker) before marking a file
  `safe`; downloads always go through signed, expiring URLs.
- Never leaks raw storage paths.

### 2.5 Identity / Auth Service
- Host accounts, login (password + optional MFA), session/JWT issuance, roles.
- Visitor identity is lightweight: verified email (magic link) or an opaque
  thread token, not a full account.

### 2.6 Notification Service
- Consumes events (`message.received`, `reply.received`).
- Sends host notifications via email/webhook/push; sends visitor "you have a reply"
  emails with a secure thread link.
- Retries with exponential backoff; dead-letter queue for failures.

### 2.7 Outbound Mail Service
- Sends actual emails (via provider API like SES/Postmark, or own MTA).
- Handles DKIM signing, bounce/complaint webhooks, and delivery status.
- Kept separate from Notification so transactional sends and inbound replies are
  independently scalable and auditable.

### 2.8 SMTP Intake Service (optional, P2)
- Listens on port 25/587, accepts mail for `inbox@domain`.
- Parses MIME, extracts body + attachments, verifies SPF/DKIM/DMARC.
- Maps `Message-ID`/`References` to existing threads; otherwise creates new.
- Feeds the same `message.received` pipeline as the Submission Service.

### 2.9 Spam / Abuse Service (optional, P2)
- Content classification, heuristics, blocklists, bayesian filter.
- Emits a spam score consumed at intake; can quarantine rather than reject.

## 3. Data & Storage

| Store | Owner | Purpose |
|---|---|---|
| PostgreSQL (per service) | Conversation, Auth, Submission | Relational data, one DB/schema per service. |
| Object Storage (S3/MinIO) | Attachment | Files, encrypted at rest, versioned. |
| Redis | Gateway, Submission | Rate limiting, sessions/cache, idempotency. |
| Message Bus (NATS/RabbitMQ) | All | Async events, decoupling, retries. |
| Search (Meilisearch/OpenSearch) | Conversation (P2) | Full-text thread search (optional). |

Rule: **database-per-service**. Services never read each other's tables; they call
APIs or consume events.

## 4. Core Data Model (simplified)

```
conversation(id, subject, status, created_at, last_activity_at)
message(id, conversation_id, direction[inbound|outbound], body, body_html,
        from_addr, to_addr, created_at, spam_score, message_id, in_reply_to)
participant(id, conversation_id, role[visitor|host], email, verified_at)
attachment(id, message_id, filename, content_type, size, storage_key,
           scan_status[pending|clean|infected], sha256)
user(id, email, password_hash, role, mfa_secret, created_at)
notification(id, user_id, message_id, channel, status, attempts)
```

## 5. Key Flows

### 5.1 Visitor sends a message + files
1. Frontend gets a captcha token and, optionally, presigned upload URLs.
2. Uploads files directly to the Attachment Service (bypasses large payloads at gateway).
3. `POST /messages` with text + attachment IDs + captcha token.
4. Submission validates, creates conversation/message, links attachments.
5. Emits `message.received` → Notification Service alerts host.

### 5.2 Host replies
1. Host opens thread in dashboard (via Gateway → Conversation).
2. `POST /conversations/{id}/replies` with body (+ attachments).
3. Conversation stores outbound message, emits `reply.created`.
4. Outbound Mail Service sends email to visitor; visitor can also view thread by token.
5. Delivery/bounce events update message status.

### 5.3 Attachment download
1. Host requests download; gateway authorizes.
2. Attachment Service checks `scan_status == clean`, returns short-lived signed URL.

## 6. Cross-Cutting Concerns

- **Auth**: JWT for hosts, opaque tokens for visitors; short TTL + refresh.
- **Rate limiting & idempotency**: at the gateway (Redis-backed).
- **Validation**: schema validation per service; reject unknown fields.
- **Observability**: structured logs, request/trace IDs propagated across services
  (OpenTelemetry), metrics per service, health/readiness endpoints.
- **Security**: TLS everywhere, secrets in a vault/manager, least-privilege DB users,
  signed URLs, malware scan, input sanitization, CSP for frontend.
- **Reliability**: at-least-once events with idempotent consumers, retries + DLQ,
  circuit breakers on downstream calls.
- **Backups**: nightly Postgres dumps + object storage replication; periodic restore drill.

## 7. Deployment Topology

### P0–P1 (single VPS)
Docker Compose: reverse proxy (Caddy/Traefik), frontend, gateway, 3–4 services,
Postgres, Redis, MinIO, ClamAV, message bus.

### P2+ (cluster)
Kubernetes (or Nomad): each service a Deployment + HPA, managed Postgres, S3,
Redis, ingress controller, cert-manager, sealed secrets, Prometheus/Grafana/Loki.

## 8. API Sketch (REST/JSON)

```
POST   /api/v1/messages                 # visitor submit (captcha)
POST   /api/v1/attachments/upload-url   # get presigned URL
POST   /api/v1/attachments/{id}/complete

POST   /api/v1/auth/login
GET    /api/v1/conversations
GET    /api/v1/conversations/{id}
POST   /api/v1/conversations/{id}/replies
PATCH  /api/v1/conversations/{id}        # label/archive/read
DELETE /api/v1/conversations/{id}

GET    /api/v1/attachments/{id}/download # signed redirect
GET    /healthz  /readyz                 # per service
```

## 9. Evolution Path (avoid premature distribution)

Start with 3 deployables: **frontend**, **core API** (submission+conversation+auth),
**worker** (notifications+outbound). Split into the full service set only when a
concrete scaling or team boundary requires it. Keep the internal contracts
(event names, API shapes) stable so splitting later is mechanical.
