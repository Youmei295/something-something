# Prototype Development Plan

Six milestones. Each is independently deployable and ends with a demoable result.
Work them in order; do not start a later milestone to escape a hard earlier one.

Legend: **Est.** = rough effort in focused days for one developer.

---

## M0 — Foundations (Est. 1–2)

**Deliverable:** repo skeleton + containers boot locally.

Tasks
- [ ] Monorepo layout (`src/frontend`, `src/backend`, `packages/`,
      `deploy/compose`).
- [ ] Go module + chi server with `/healthz` and `/readyz`.
- [ ] Next.js app skeleton with Tailwind + a placeholder submit page.
- [ ] `docker-compose.yml`: web, api, postgres, minio.
- [ ] `deploy/compose/.env.example` documenting every required variable.
- [ ] Makefile (or Taskfile) with `make dev`, `make up`, `make migrate`, `make test`.
- [ ] `docs/prototype/` committed; ADRs referenced in code comments where relevant.

Acceptance
- `make up` brings up the stack; `curl localhost/healthz` returns 200.
- Placeholder page renders in the browser.

---

## M1 — Visitor can submit a message (Est. 2–3)

**Deliverable:** a message submitted from the public page is stored in Postgres.

Tasks
- [ ] DB schema v1: `conversations`, `messages`, `users` (migration via goose).
- [ ] `POST /api/v1/messages` — validate (name, email, subject, body), store.
- [ ] Public Next.js form wired to the endpoint, with client + server validation.
- [ ] Honeypot field + basic per-IP rate limit (in-memory for now).
- [ ] Confirmation page with the conversation reference.
- [ ] Integration test: submit → row exists.

Acceptance
- A real submission appears in the DB with correct fields.
- Invalid/empty input is rejected with clear errors.
- Honeypot blocks naive bots.

---

## M2 — Host inbox (Est. 2–3)

**Deliverable:** host logs in and reads messages in a dashboard.

Tasks
- [ ] Host user seed command (env-provided email/password, argon2id hash).
- [ ] Session-based login/logout (`POST /api/v1/auth/login`, httpOnly cookie).
- [ ] Protected routes + middleware rejecting unauthenticated access.
- [ ] `GET /api/v1/conversations` (list) and `GET /api/v1/conversations/{id}`.
- [ ] Dashboard: list view + detail view, unread indicators.
- [ ] Mark-as-read.
- [ ] Tests for auth middleware and list/detail authorization.

Acceptance
- Unauthenticated users cannot reach the dashboard or API.
- The message from M1 is visible and readable in the dashboard.
- Session survives a page refresh and expires server-side.

---

## M3 — Attachments (Est. 2–3)

**Deliverable:** a visitor can attach one file; the host can download it safely.

Tasks
- [ ] MinIO container + bucket bootstrap.
- [ ] `POST /api/v1/attachments/upload-url` → presigned PUT (5 MB, image/PDF only).
- [ ] Client uploads directly to MinIO, then includes the attachment ID in submit.
- [ ] `attachment` table: filename, content_type, size, storage_key, sha256, status.
- [ ] `GET /api/v1/attachments/{id}/download` → authorize + short-lived signed URL.
- [ ] Reject oversized/disallowed types server-side (never trust the client).
- [ ] Cleanup job for orphaned uploads.

Acceptance
- Upload + attach + download works end to end.
- A disallowed type or >5 MB upload is rejected by the API.
- The bucket is not publicly readable.

---

## M4 — Replies + email (Est. 2–3)

**Deliverable:** host replies, visitor gets an email; worker handles async sending.

Tasks
- [ ] `Mailer` interface + Postmark/SES implementation + a console/dev implementation.
- [ ] Postgres-backed job table; worker polls with `FOR UPDATE SKIP LOCKED`.
- [ ] `POST /api/v1/conversations/{id}/replies` stores an outbound message + enqueues.
- [ ] Worker sends the reply email with a tokenized thread link.
- [ ] Email the host on new inbound message (notification job).
- [ ] Retry with backoff; mark jobs failed after N attempts.
- [ ] Visitor-facing minimal thread page (tokenized) to view the reply.

Acceptance
- Replying delivers an email to the visitor's address within ~30s.
- Host receives a new-message notification.
- A failing send retries and is visible as failed, not silently lost.

---

## M5 — Ship it (Est. 1–2)

**Deliverable:** the prototype runs on a real domain over HTTPS.

Tasks
- [ ] VPS provisioning notes (Docker, firewall, non-root deploy user).
- [ ] Caddy config for automatic TLS + reverse proxy to web/api.
- [ ] GitHub Actions: test on PR; on `main` build → push GHCR → deploy over SSH.
- [ ] Secrets via host env file (or SOPS); document required secrets.
- [ ] Postgres + MinIO persistent volumes; nightly `pg_dump` to disk.
- [ ] Smoke test after deploy (hits `/healthz` and submits a test message).
- [ ] Basic runbook: restart, logs, rollback.

Acceptance
- Push to `main` deploys automatically; smoke test passes.
- Site loads over HTTPS with a valid certificate.
- A rollback to the previous image is documented and tested once.

---

## M6 — Polish & prototype review (Est. 1)

**Deliverable:** the prototype is presentable and the stack decisions are confirmed.

Tasks
- [ ] Error states, empty states, loading states in the UI.
- [ ] Accessibility pass on the public form (labels, focus, contrast).
- [ ] Log redaction check (no email bodies/secrets in logs).
- [ ] Write `05-prototype-review.md`: what worked, what to change, ADR revisions.
- [ ] Confirm or amend the frozen stack in `02-tech-stack.md`.

Acceptance
- The full DoD checklist in `01-scope.md` is satisfied.
- A documented decision exists for every stack item carried into v0.2.

---

## Cross-Milestone Habits

- One short-lived branch per milestone; squash-merge to `main`.
- Add a test with every behavior change; no milestone merges with red tests.
- Keep `docs/` current — decisions live in the repo, not in chat.
- Tag every milestone: `v0.1-M0` … `v0.1`.
