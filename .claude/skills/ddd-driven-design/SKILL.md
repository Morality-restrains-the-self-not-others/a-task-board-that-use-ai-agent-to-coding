---
name: ddd-driven-design
description: >
  Domain-Driven Design enforcement for task2app backend development. Triggers on
  domain-layer code, aggregates, entities, value objects, domain events, repositories,
  or any backend feature work. MUST be invoked alongside superpowers 7-step workflow
  for all server-side development.
  Triggers on: Saas_project/**/domain/**, backend feature, Django model changes,
  business logic, service-layer code, or any task mentioning 领域/domain/aggregate/entity.
license: MIT
metadata:
  version: "1.0.0"
  source: .ai/03_technical_implementation/09_domain_driven_design.md
  ci_gate: scripts/ci/check_ddd_bdd_compliance.py
  ci_workflow: .github/workflows/ddd-bdd-compliance.yml
---

# DDD-Driven Design

Domain-Driven Design is **mandatory** for all server-side development in this project.
CI enforces it via `check_ddd_bdd_compliance.py` — domain-layer files MUST NOT import
ORM, Kafka, cloud SDKs, or other infrastructure directly.

This skill provides the **design-time guidance** that complements the CI's static checks.
It integrates into the superpowers 7-step workflow so DDD thinking is applied at every
stage, not just caught at the gate.

---

## When to Invoke This Skill

- Any backend feature work in `Saas_project/`, `taskAiProvider/`, `Saas_email/`
- Creating or modifying files under `**/domain/**`
- Designing new business capabilities
- Refactoring existing service code
- Whenever the superpowers brainstorming skill is invoked for a backend task

Invoke with: `Skill(skill="ddd-driven-design")` at the START of each workflow step.

---

## Core DDD Rules (from `.ai/` rules)

### DDD-01: Domain Model First
Business logic is organized around domain models (aggregates, entities, value objects),
NOT around database tables or framework structures. Design the model in isolation before
choosing persistence.

### DDD-02: Domain Events for Cross-Aggregate Communication
Inter-aggregate, cross-module collaboration uses domain events (publish/consume), not
direct method calls. Within a single aggregate, direct method calls are fine.

### DDD-03: Infrastructure Isolation (CI-enforced)
Domain layer must NOT directly import:
- `django.db`, `django.db.models`, `django.contrib.auth.models`
- `kafka`, `confluent_kafka`
- `boto3`, `botocore`, `oss2`, `aliyunsdkcore`
- `pymysql`, `sqlalchemy`
- `redis`, `celery`

Infrastructure access goes through interfaces/ports defined in the domain layer,
with adapters in `infrastructure/` or `adapters/`.

### DDD-04: Clear Bounded Contexts
Modules are divided by business domain. Logic within a domain is cohesive; domains
communicate through well-defined interfaces or domain events.

### DDD-05: Aggregate Design
Each aggregate has a root entity that enforces invariants. External code references
the aggregate by its root ID only. Nested entities are accessed through the root.

### DDD-06: Repository Pattern
Data access is abstracted behind repository interfaces in the domain layer.
Repository implementations live in the infrastructure/adapter layer.

### DDD-07: Single-Service Data Ownership (process boundary)
A database (or logical DB) and each table may be accessed directly by **only one
owner service**. Other services must not share connection strings, reuse Models/DAOs,
or run side-channel SQL (read or write). If multiple services need the same data,
use **service forwarding** (owner API / RPC / domain events) or **migrate the table**
so a single owner remains. See
`.ai/01_project_constraints/19_single_service_data_ownership.md`.

---

## Integration with Superpowers 7-Step Workflow

### Step 1 — Brainstorming: DDD Discovery

Apply DDD thinking during the design phase, BEFORE any code is written.

**DDD Discovery Checklist:**

- [ ] **Identify Bounded Contexts** — What business domains are involved? Where are the boundaries? Draw a context map if multiple contexts interact.
- [ ] **Identify Aggregates** — What are the transactional consistency boundaries? Which entity is the aggregate root?
- [ ] **Identify Domain Events** — What significant business occurrences cross aggregate or context boundaries? Name them in past tense: `OrderPlaced`, `PaymentReceived`, `UserRegistered`.
- [ ] **Design Ubiquitous Language** — Are the terms used in the design matching what domain experts use? Document non-obvious terms.
- [ ] **Design Ports (Interfaces)** — What does the domain need from the outside world? Define repository interfaces, event publisher interfaces, external service interfaces as abstract contracts.
- [ ] **Decide persistence strategy** — What data does each aggregate need? Is it a full ORM aggregate, event-sourced, or document-backed? Note this for the plan phase.

**Output:** These decisions go into the design doc under a "Domain Model" section before
proceeding to writing-plans.

### Step 2 — Worktrees: No DDD-specific actions (standard isolation workflow)

### Step 3 — Plans: DDD Structure Validation

When writing the implementation plan, validate DDD structure:

- [ ] **File layout follows DDD layering:**
  ```
  {module}/
    domain/           # Aggregates, entities, value objects, domain services, events, repository interfaces
    application/      # Use cases / application services, DTOs
    infrastructure/   # Repository implementations, external API adapters, messaging adapters
    interfaces/       # API views, serializers (Django REST framework)
  ```
- [ ] **No infrastructure imports planned for domain-layer files**
- [ ] **Each task creates domain-layer files BEFORE infrastructure/adapter files**
- [ ] **Repository interfaces defined in domain BEFORE implementations in infrastructure**
- [ ] **Domain event classes defined in domain BEFORE publishers/consumers in infrastructure**
- [ ] **Test files placed in correct layers:** `tests/domain/` for domain logic, `tests/infrastructure/` for adapters

### Step 4 — Build: DDD Enforcement

During implementation, enforce DDD rules:

- [ ] **Domain files contain ZERO infrastructure imports** — This is the #1 CI failure. Before committing any domain file, scan imports.
- [ ] **Aggregate roots enforce invariants** — No external code can put an aggregate in an invalid state. Business rules live in the aggregate, not in services.
- [ ] **Value objects are immutable** — If it represents a value (Money, Email, PhoneNumber), it should be immutable and equality-based.
- [ ] **Entities have identity** — If it has a lifecycle and identity that persists across changes, it's an entity with a stable ID.
- [ ] **Domain services are stateless** — They orchestrate aggregates and domain logic but hold no state themselves.
- [ ] **Application services coordinate** — They load aggregates via repositories, call domain logic, and commit changes. They do NOT contain business rules.
- [ ] **Run CI compliance check locally before committing:**
  ```bash
  python scripts/ci/check_ddd_bdd_compliance.py
  ```

### Step 5 — TDD: DDD Test Patterns

When writing tests in the DDD context:

- [ ] **Aggregate tests** — Test aggregate behavior and invariant enforcement. Use the aggregate root's public API only. A test should look like: arrange events/state → call command → assert events/state.
  ```python
  # tests/domain/order/test_order_aggregate.py
  def test_cannot_add_item_to_shipped_order():
      order = Order.restore(id=1, status="SHIPPED", items=[])
      with pytest.raises(DomainError, match="Cannot modify shipped order"):
          order.add_item(ProductId(5), Quantity(2))
  ```

- [ ] **Value object tests** — Test equality, immutability, validation, and factory methods.
  ```python
  # tests/domain/shared/test_money.py
  def test_money_equality():
      assert Money(100, "CNY") == Money(100, "CNY")
      assert Money(100, "CNY") != Money(100, "USD")

  def test_money_cannot_be_negative():
      with pytest.raises(ValueError):
          Money(-1, "CNY")
  ```

- [ ] **Domain event tests** — Verify events carry correct data and are raised at the right time.
- [ ] **Repository tests (infrastructure)** — Test against real database (SQLite :memory: or test DB), not mocks. These go in `tests/infrastructure/`.
- [ ] **Use InMemory implementations for domain-layer testing** — `InMemoryEventPublisher`, `InMemoryEmailSender` for isolating domain tests from infrastructure.
- [ ] **Strictly follow the red-green-refactor cycle** — Domain logic tests first, then implementation.

### Step 6 — Review: DDD Compliance Audit

When reviewing code, apply the DDD lens:

- [ ] **Import scan** — Does any domain file import from `FORBIDDEN_DOMAIN_IMPORT_PREFIXES`?
- [ ] **Aggregate boundary check** — Are invariants enforced in the aggregate root? Can external code bypass the root?
- [ ] **Event flow check** — Are cross-aggregate operations using domain events, not direct calls?
- [ ] **Repository abstraction** — Does the domain layer depend on interfaces, not concrete implementations?
- [ ] **Layering check** — Are application services free of business rules? Are domain services stateless?
- [ ] **Value object usage** — Are Money, Email, PhoneNumber, etc. modeled as value objects instead of primitives?
- [ ] **Run compliance script on the diff:**
  ```bash
  python scripts/ci/check_ddd_bdd_compliance.py
  ```

**Block on:** Domain files importing infrastructure, business logic in application layer,
missing repository interfaces, cross-aggregate direct calls.

### Step 7 — Ship: DDD Gate

Before merging/shipping:

- [ ] CI DDD/BDD compliance workflow passes (`.github/workflows/ddd-bdd-compliance.yml`)
- [ ] All domain-layer tests pass (no infrastructure dependencies)
- [ ] All infrastructure tests pass
- [ ] Domain events are properly wired (publisher and consumer both present)

---

## DDD Layering Reference

```
┌──────────────────────────────────────────────┐
│  interfaces/ (API views, serializers, CLI)    │
│  Depends on: application/                     │
├──────────────────────────────────────────────┤
│  application/ (use cases, app services, DTOs) │
│  Depends on: domain/                          │
├──────────────────────────────────────────────┤
│  domain/ (aggregates, entities, VOs, events,  │
│           repository interfaces, domain svcs) │
│  Depends on: NOTHING external                 │
├──────────────────────────────────────────────┤
│  infrastructure/ (repo impls, adapters,       │
│                    event bus, external APIs)   │
│  Depends on: domain/ (for interfaces)         │
└──────────────────────────────────────────────┘
```

The key rule: **dependencies point inward. `domain/` has zero external dependencies.**

---

## Aggregate Design Checklist

Use this when designing or reviewing an aggregate:

- [ ] **Single root entity** — One entity per aggregate is the root; all external access goes through it
- [ ] **ID-based references** — Other aggregates are referenced by ID, not by object reference
- [ ] **Invariants enforced** — The aggregate root ensures consistency of the entire aggregate boundary
- [ ] **Small aggregates** — When in doubt, make aggregates smaller. A large aggregate is a design smell
- [ ] **One aggregate per transaction** — A single transaction modifies exactly one aggregate instance
- [ ] **Event on state change** — Significant state changes raise a domain event

## Domain Event Checklist

Use this when designing domain events:

- [ ] **Named in past tense** — `OrderPlaced`, not `PlaceOrder` or `OrderPlaceEvent`
- [ ] **Carries relevant data** — Aggregate ID, relevant values, timestamp; minimal and immutable
- [ ] **Published by the aggregate** — The aggregate records the event; infrastructure publishes it
- [ ] **Cross-aggregate** — Events are for communication BETWEEN aggregates, not within
- [ ] **Event handler is idempotent** — A consumer may receive the same event more than once
- [ ] **Intent → event → MQ** — Every accepted business intent that changes facts or triggers cross-boundary side effects has a corresponding domain event published to the message queue (via EventBus port). Document the mapping in `docs/intents/*.intent.md`. Pure queries may omit events only with an explicit written exception.

## Common DDD Anti-Patterns

| Anti-Pattern | Why It Fails | Correct Approach |
|---|---|---|
| Anemic domain model — entities with only getters/setters, logic in services | Domain logic leaks out, aggregates don't enforce invariants | Put behavior and validation in the aggregate/entity |
| Direct ORM imports in domain | CI fails; domain coupled to persistence | Repository interface in domain, ORM implementation in infrastructure |
| Fat service with all business logic | Transaction script, not DDD; impossible to test domain in isolation | Distribute logic into aggregates, entities, value objects |
| Primitive obsession — Money as `Decimal`, Email as `str` | No validation, no behavior, scattered conversion logic | Value objects: `Money(amount, currency)`, `Email(value)` |
| Cross-aggregate direct calls — `order_service` calling `payment_service.process()` | Tight coupling, no eventual consistency option | Domain events: `OrderConfirmed` → payment consumer reacts |
| Aggregate as data dump — one giant aggregate with everything | Performance, concurrency conflicts, bloated invariants | Split into smaller aggregates referenced by ID |
| Missing repository interface — domain imports Django ORM directly | CI fails; domain coupled to framework | Abstract interface in domain, implementation in infrastructure |
| Domain logic in views/controllers | Business rules scattered, untestable in isolation | Application service calls domain, API view calls application service |
| Skipping domain events for "simple" cross-module calls | Each "simple" call becomes a coupling anchor | Default to events; make synchronous calls the exception |
| **Business intent with no MQ event** — command succeeds but never publishes | Downstream (email/SSE/other aggregates) couples via sync calls; no audit trail of the intent | Define past-tense event; publish via EventBus after state change; map in intent doc |
| Application service contains business validation | Business rules live outside the domain, duplicated across services | Validate in aggregate/entity/value object constructors |

---

## Quick Reference: DDD Self-Check Before Commit

Run through this before committing any backend code:

| # | Check | If Fail |
|---|-------|---------|
| 1 | No `django.db` / `boto3` / `kafka` / `redis` / `celery` import in `**/domain/**/*.py` | Move to adapter, define interface in domain |
| 2 | Aggregate root enforces its invariants | Add invariant checks to aggregate methods |
| 3 | Cross-aggregate communication uses events | Replace direct call with domain event publish/consume |
| 3b | Every business intent publishes a corresponding MQ event (or documented exception) | Add event + EventBus publish; update `docs/intents/` mapping |
| 4 | Repository interface in domain, implementation in infrastructure | Split file, inject implementation |
| 5 | Value objects for Money, Email, Phone, etc. | Create VO class with validation |
| 6 | `python scripts/ci/check_ddd_bdd_compliance.py` passes | Fix violations, re-run |

---

## Project-Specific Conventions

### Django without ORM coupling
- Repository interfaces in `domain/` define `save()`, `get_by_id()`, `find_by_criteria()`
- Repository implementations in `infrastructure/` use Django ORM internally but return domain objects
- Domain objects are plain Python classes, NOT Django models

### Event flow
- Aggregates record events: `self._events.append(OrderPlaced(order_id=self.id, ...))`
- Application service collects events after aggregate operations
- Infrastructure `EventPublisher` sends events to Kafka or equivalent message bus
- Test with `InMemoryEventPublisher` for domain-layer tests
- **Rule:** when a business intent is accepted, the corresponding event MUST be published to the queue (see `.ai/08_prompt_management/01_intent_driven_development.md`)

### CI Gate
- `scripts/ci/check_ddd_bdd_compliance.py` runs on every push and PR
- Pre-commit hook version runs `--staged` mode
- Both check domain layer for forbidden imports AND verify test co-evolution
