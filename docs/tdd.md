# Technical Design Document — Bridge

## Technology Stack

- **Language**: Go
- **Architecture**: DDD + event-driven
- **Messaging**: RabbitMQ
- **Persistence**: PostgreSQL (sqlc)
- **State**: statekit (state machines)
- **Logging**: felixgeelhaar/bolt
- **Policy**: OPA (Rego)
- **Frontend**: Astro + Vue + TypeScript
- **Infra**: Docker, Kubernetes (optional), Terraform
- **Observability**: OpenTelemetry

---

## High-Level Architecture

CLI / CI
→ Orchestrator
→ Policy Engine
→ Workflow Engine
→ Agent Runners
→ Audit / Telemetry

---

## Core Bounded Contexts

### Workflow

- WorkflowDefinition
- WorkflowRun
- Step
- State transitions via statekit

### Governance

- PolicyBundle
- Approval
- AuditEvent

### Agents

- Agent
- ToolBinding (MCP)
- Capability model

### Analytics

- TelemetryEvent
- Metrics aggregation

---

## Execution Model

- Orchestrator schedules steps
- Steps are isolated, idempotent
- State machine controls transitions
- Policies evaluated pre/post execution

---

## Logging & Observability

- Structured logs via **Bolt**
- Context-aware logging (trace/run/step IDs)
- OTEL tracing across services
- Metrics for cost, latency, outcomes

---

## Security

- Zero-trust
- No secrets stored by default
- BYO LLM keys
- Signed workflows & policies
- Immutable audit logs

---

## Deployment Model

- Default: local + CI
- Optional: hosted control plane
- Execution remains customer-owned
