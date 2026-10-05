# Prototype (v0.1) — Overview

**Goal:** build the smallest working end-to-end system that proves the core idea and
**freezes the tech stack** that later versions will build on.

The prototype is deliberately small. It is not the microservice architecture from
`../02-architecture.md`; it is the seed from which that architecture grows.

## What "done" means for the prototype

A visitor can:

1. open a public page,
2. submit a message (and optionally one small file),
3. see a confirmation.

The host can:

4. log in,
5. see the message in a list,
6. open it and read the message + download the file,
7. reply, and
8. the visitor receives the reply by email.

If those eight steps work reliably on a real server over HTTPS, the prototype is done.

## Why it matters beyond the demo

- It validates the core user loop before investing in scale.
- It forces hand-on work with the stack: containers, TLS, DB, storage, email, CI.
- It produces a **decision record** (`02-tech-stack.md`) so version 1.0 does not
  re-litigate technology choices.
- It establishes the internal contracts (message model, events, API shapes) that the
  future microservices will reuse.

## Documents in this folder

| Doc | Purpose |
|---|---|
| [01-scope.md](./01-scope.md) | In-scope / out-of-scope for the prototype. |
| [02-tech-stack.md](./02-tech-stack.md) | ADR-style decisions that bind later versions. |
| [03-development-plan.md](./03-development-plan.md) | Milestones, tasks, acceptance criteria. |
| [../04-roadmap-v1.md](../04-roadmap-v1.md) | Steps from prototype → v1.0. |
| [05-running-services.md](./05-running-services.md) | Run each service separately (monolith vs microservice modes). |

## Guiding principles

1. **Working beats complete.** Ship the loop, then harden.
2. **Two services, not nine.** Split only when a boundary hurts.
3. **Freeze the stack, stay flexible on internals.** Decisions here are contracts.
4. **Operate it for real.** Deploy to a VPS with HTTPS, not just localhost.
5. **Every milestone is deployable.** No long-lived unmerged branches.
