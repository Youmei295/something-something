# Docs

Planning documents for the "Inbox" project (visitor → host messaging server, microservices).

### Get started
- [Startup Guide](./STARTUP.md) — run the app (Docker monolith, microservices, or local processes), configuration, troubleshooting.

### Vision & design
1. [Proposal](./01-proposal.md) — problem, goals, scope, phases, risks.
2. [Architecture Plan](./02-architecture.md) — services, data, flows, deployment.
3. [Tech Stack](./03-tech-stack.md) — recommended stack, minimal P0 stack, alternatives.

### Prototype (v0.1) — start here
- [Prototype Overview](./prototype/README.md) — goal, definition of done, principles.
- [Scope](./prototype/01-scope.md) — in/out of scope, minimal architecture.
- [Tech Stack Decisions (ADR)](./prototype/02-tech-stack.md) — **binding** stack choices.
- [Development Plan](./prototype/03-development-plan.md) — milestones M0–M6 + acceptance criteria.
- [Running Services](./prototype/05-running-services.md) — run the services separately to mimic the microservice architecture.

### Roadmap
- [Prototype → v1.0](./04-roadmap-v1.md) — versions v0.1–v1.0, compatibility rules.

## One-line pitch

A self-hosted inbox server where visitors send messages + files to a host, the host
replies, and both sides get a clean threaded experience — built as independently
deployable microservices for learning real server operations.
