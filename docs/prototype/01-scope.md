# Prototype Scope (v0.1)

## In Scope

### Visitor side
- Public submission page: name, email, subject, message body.
- One optional file attachment, max **5 MB**, allowed types: images + PDF.
- Simple CAPTCHA or honeypot (one of them is enough for the prototype).
- Confirmation screen after submit.
- Visitor receives the host's reply by email (no visitor account, no thread UI yet).

### Host side
- Host login (single host account; seeded or created via CLI/env).
- Inbox list: subject, sender, date, unread indicator.
- Conversation view: original message + attachments.
- Reply box that sends an email back to the visitor.
- Download attachment (served through the app, not a public path).

### Platform
- Runs on a single VPS with Docker Compose.
- HTTPS via reverse proxy with automatic certificates.
- Postgres + object storage + outbound email provider.
- Basic logging and a health endpoint.
- CI that builds and deploys on push to `main`.

## Out of Scope (deferred)

| Deferred | Target version |
|---|---|
| SMTP inbound (real `inbox@domain`) | v0.3 (P2) |
| Spam scoring / blocklists | v0.3 |
| Malware scanning (ClamAV) | v0.2 |
| Multiple hosts / teams / roles | v0.4 |
| Visitor thread-view web page | v0.2 |
| Full-text search | v0.4 |
| MFA, password reset, audit log | v0.3–v0.4 |
| Kubernetes, autoscaling | v1.0 |
| Webhooks, push notifications | v0.4 |
| Rate limiting beyond a simple IP cap | v0.2 |
| Attachment streaming for large files | v0.3 |
| Backups & restore drills | v0.2 |

## Constraints

- **Single developer.** Keep the deployable count low.
- **Single small VPS.** Memory budget: target services total < 2 GB RAM.
- **No paid infra required** beyond a domain and an email provider free tier.
- **Time-box:** prototype should be reachable in a few focused weekends.

## Prototype architecture (deliberately minimal)

```
        Visitor / Host browser
                 │ HTTPS
                 ▼
           ┌───────────┐
           │  Caddy    │  (TLS + reverse proxy)
           └─────┬─────┘
                 │
        ┌────────┴─────────┐
        ▼                  ▼
  ┌───────────┐      ┌───────────┐
  │ Next.js   │      │  Go API   │
  │ frontend  │─────▶│  service  │
  └───────────┘      └─────┬─────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
        ┌──────────┐  ┌────────┐  ┌──────────┐
        │ Postgres │  │ MinIO  │  │ Email    │
        └──────────┘  └────────┘  │ provider │
                                  └──────────┘
              ▲
        ┌─────┴──────┐
        │ Go worker  │  (async: send email, retries)
        └────────────┘
```

Three deployables: **web**, **api**, **worker**. No message bus in the prototype —
the worker uses a Postgres-backed job table. The bus is introduced in v0.2.

## Definition of Done (checklist)

- [ ] Visitor can submit a message with a file from a phone and desktop.
- [ ] Host sees it within 5 seconds and gets an email notification.
- [ ] Host reply lands in the visitor's email inbox.
- [ ] Attachment downloads only through the app.
- [ ] Everything is reachable over HTTPS on a real domain.
- [ ] CI deploys automatically and a smoke test passes after deploy.
- [ ] The stack decisions in `02-tech-stack.md` are all implemented or explicitly revised.
