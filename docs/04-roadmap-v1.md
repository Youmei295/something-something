# Roadmap: Prototype → v1.0

Each version is a shippable increment. Versions map to the phases in
`../01-proposal.md` (P0–P3) and the service split in `../02-architecture.md`.

```
v0.1 prototype ─▶ v0.2 harden ─▶ v0.3 real email ─▶ v0.4 scale-out ─▶ v1.0 stable
   (P0)              (P1)            (P2)               (P2)            (P3)
```

---

## v0.1 — Prototype (P0)

**Theme:** prove the loop, freeze the stack.
**Features:** web submit + 1 attachment, host login/inbox/reply, outbound email,
1 VPS + HTTPS + CI.
**Architecture:** web + api + worker, Postgres, MinIO, no message bus.
**Exit criteria:** the DoD checklist in `prototype/01-scope.md` passes; ADRs confirmed.

---

## v0.2 — Hardening & correct plumbing (P1)

**Theme:** make the prototype trustworthy and introduce the event backbone.

Planned work
- Introduce **NATS JetStream**; migrate the job table to events.
- **ClamAV** malware scanning; attachments downloadable only when `clean`.
- **Redis** for rate limiting, sessions, idempotency keys.
- Host email **verification**; visitor email **verification** (reduce spoofing).
- **Backups**: automated Postgres dumps + MinIO replication, first restore drill.
- **Observability**: Prometheus + Grafana + Loki, OpenTelemetry traces.
- Structured **audit log** for host actions.
- Visitor thread-view page hardened (token expiry, revocation).
- **First service split**: extract **Attachment Service** and/or **Notification
  Service** behind gRPC when it clearly reduces coupling.

Exit criteria
- Attachments are scanned before download; scan failures are visible.
- One restore drill completed and documented.
- Metrics/logs/traces visible in dashboards.

---

## v0.3 — Real email & trust (P2)

**Theme:** accept actual mail and defend the inbox.

Planned work
- **SMTP intake service** (`inbox@domain`) parsing MIME → same message model.
- SPF/DKIM/DMARC verification; header threading (`Message-ID`/`References`).
- **Spam/abuse service**: scoring, blocklists, quarantine queue.
- **Reply-by-email**: the host can reply from their mail client and it threads back.
- Bounce/complaint handling from the email provider.
- MFA for the host; password reset; session management UI.
- Extract **Conversation Service** and **Submission Service** as independent deploys.

Exit criteria
- An external email sent to `inbox@domain` appears as a conversation.
- Spam is quarantined, not shown by default.
- End-to-end reply-by-email round trip works.

---

## v0.4 — Multi-host & scale-out (P2)

**Theme:** more than one operator; more than one server if needed.

Planned work
- Multiple hosts, teams, roles/permissions, per-host inboxes and quotas.
- Full-text **search** (Meilisearch or Postgres FTS) across conversations.
- **Webhooks** and additional notification channels.
- Extract remaining services: **Identity**, **Outbound Mail**, plus **API Gateway**
  as a separate deployable.
- Load testing; define SLOs and error budgets.
- Attachment streaming / large-file support (multipart, resumable).

Exit criteria
- Two hosts use the system without seeing each other's data.
- Documented load test meets the SLOs.
- All services independently deployable.

---

## v1.0 — Stable platform (P3)

**Theme:** production-grade, reproducible, observable, recoverable.

Planned work
- **Kubernetes (k3s) or Nomad**: per-service Deployments + autoscaling, rolling
  deploys, self-healing.
- **GitOps** (Argo CD) replacing SSH-based deploy.
- **IaC**: Terraform for infra, Ansible for host config.
- **Vault** (or SOPS) for secrets; cert-manager for TLS.
- Zero-downtime migrations and deploys; documented RTO/RPO.
- Security review: dependency scanning, SBOM, penetration test checklist.
- Public API docs (OpenAPI), versioning policy, deprecation policy.
- Onboarding docs: local dev in < 15 minutes, architecture map, runbooks.

Exit criteria
- Deploys are zero-downtime and rollback is one command.
- Backup/restore meets the documented RTO/RPO in a drill.
- A new contributor can run the stack locally and ship a change using the docs alone.
- All v1.0 SLOs are met for 30 consecutive days.

---

## Versioning & Compatibility Rules

- **Public API:** semver; `/api/v1` stable for the life of the major version.
- **Events:** versioned subjects/types (`message.received.v1`); additive-only within
  a major version.
- **DB migrations:** forward-only and backward-compatible with the previous app
  version (expand/contract pattern) to allow zero-downtime deploys from v1.0.
- **ADR changes:** any frozen stack change requires a new ADR with a migration note.

## Milestone → Version Map

| Version | Proposal phase | Architecture stage |
|---|---|---|
| v0.1 | P0 prototype | web + api + worker |
| v0.2 | P1 MVP/hardening | + NATS, ClamAV, Redis, first split |
| v0.3 | P2 email & trust | + SMTP intake, spam, conversation/submission split |
| v0.4 | P2 scale-out | full service set + gateway |
| v1.0 | P3 operations | Kubernetes + GitOps + IaC + SLOs |
