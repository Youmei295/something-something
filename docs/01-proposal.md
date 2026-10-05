# Proposal — "Inbox" (working title)

## 1. Summary

A self-hosted messaging platform where **visitors** can send messages and files to the
**host** of the server, and the **host** can reply. Conceptually it is a private,
programmable email/inbox server with a modern web frontend.

The backend is split into small **microservices** so each concern (receiving mail,
storing threads, handling files, notifying, replies) can be developed, scaled, and
deployed independently. This makes the project a realistic playground for practicing
server operations, container orchestration, and distributed-system patterns.

## 2. Problem / Motivation

- Generic contact forms are one-way: a visitor submits, and gets no clean threaded reply.
- Running a real mail server (Postfix/IMAP) is powerful but heavy and unforgiving.
- The goal here is a middle ground: a controlled **inbox server** with an HTTP/SMTP
  intake, a host dashboard, threaded replies, and file attachments — all self-hosted.
- Secondary goal: practice managing a server, containers, networking, TLS, backups,
  monitoring, and CI/CD.

## 3. Goals

1. Visitors can send a message + attachments to a host without creating an account.
2. The host sees all incoming messages in one dashboard, grouped into conversations.
3. The host can reply; the reply is delivered back to the visitor (email and/or web link).
4. Attachments are stored safely, size-limited, scanned, and served via signed URLs.
5. Anti-spam: captcha, rate limiting, content filtering, optional allow/deny lists.
6. Everything runs as containers and can be deployed on a single VPS or a small cluster.
7. Clear separation of concerns so services can be replaced or scaled individually.

## 4. Non-Goals (v1)

- Full IMAP/POP3 mailbox compatibility.
- Building a general-purpose email client.
- Multi-tenant SaaS billing.
- Mobile native apps (responsive web first).

## 5. Target Users / Personas

| Persona | Need |
|---|---|
| Visitor | Send a question, report, or files to the host quickly, no signup. |
| Host (operator) | One inbox, threaded replies, notifications, attachment access. |
| Admin (future) | Manage multiple hosts, quotas, abuse reports. |

## 6. Core User Stories

- As a visitor, I can open a public page, enter my email/message, attach files, and send.
- As a visitor, I receive a link to view the thread and any reply.
- As the host, I am notified (email/webhook) when a new message arrives.
- As the host, I can read, reply, label, archive, and delete conversations.
- As the host, I can download attachments and know they were scanned.
- As the system, I reject spam, oversized, or malicious uploads.

## 7. Two Ingestion Modes

1. **API / Web form** — the primary path. Frontend calls the Submission Service.
2. **SMTP intake** — optional advanced path. An SMTP service accepts real email at
   `inbox@yourdomain`, parses MIME, and feeds the same pipeline. Requires MX records,
   SPF/DKIM/DMARC.

Both modes converge on the same internal message model, so the rest of the system does
not care how a message arrived.

## 8. Scope by Phase

| Phase | Deliverable |
|---|---|
| P0 — Prototype | Web form → store message → host sees it (monolith-ish, 2 services). |
| P1 — MVP | Threads, replies by email, attachments, auth, notifications, rate limit. |
| P2 — Hardening | SMTP intake, spam filtering, malware scan, audit log, backups. |
| P3 — Scale/Ops | K8s, autoscaling, observability, multi-host, quotas. |

## 9. Success Metrics

- A visitor can send a message + 10 MB attachment in < 3s (p95).
- Host reply delivered to visitor email in < 30s (p95).
- Zero unhandled malicious attachments reaching the host download.
- 99.5% uptime on a single-node deployment.

## 10. Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Spam / abuse | Captcha, rate limits, content filter, blocklist, email verification. |
| Malware in uploads | ClamAV scan before the file is marked downloadable. |
| Email deliverability | Use a transactional provider or properly configure SPF/DKIM/DMARC. |
| Microservice overkill | Start with 3–4 services; split only when justified. |
| Data loss | Postgres backups + object storage versioning + restore drills. |
| Cost/complexity | Single VPS with Docker Compose for P0–P1. |
