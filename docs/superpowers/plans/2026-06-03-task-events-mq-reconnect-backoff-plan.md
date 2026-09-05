# task-events MQ Reconnect Backoff Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent task-events CPU spin when Redis/Kafka is unreachable; fail-fast at startup; expose MQ readiness via split health probes.

**Architecture:** Shared exponential backoff in `taskEvents/broker`; startup `PingWithRetry` in runner; runtime backoff in redis/kafka subscribe loops; `/api/health/ready` reflects `ConnectionState`; runAll uses liveness/readiness split probe.

**Tech Stack:** Go, redis/go-redis, segmentio/kafka-go, runAll YAML

---

### Task 1: Broker backoff + connection state

**Files:**
- Create: `taskEvents/broker/retry.go`, `connection_state.go`, `ping.go`
- Test: `taskEvents/broker/retry_test.go`, `ping_test.go`

- [x] Implement Backoff, RateLimitedLogger, ConnectionState, Ping/PingWithRetry
- [x] Unit tests pass: `go test ./broker/...`

### Task 2: Wire brokers + runner

**Files:**
- Modify: `taskEvents/broker/redis.go`, `kafka.go`, `factory.go`
- Modify: `taskEvents/consumer/runner.go`
- Modify: `taskEvents/integration/harness.go`

- [x] Runtime backoff on connection errors
- [x] Startup PingWithRetry before Subscribe
- [x] NewHandle returns Port + Conn

### Task 3: Health readiness + runAll

**Files:**
- Modify: `taskEvents/server/health.go`
- Test: `taskEvents/server/health_test.go`
- Modify: `runAll.yaml` (18 task-events services)

- [x] `/api/health/ready` returns 503 when MQ disconnected
- [x] runAll liveness_url + readiness url

### Task 4: Domain + value stream

**Files:**
- Create: `taskEvents/domain/backoff_policy.go`
- Modify: `value-stream.yaml`

- [x] BackoffPolicy + ConnectionHealth value objects
- [x] New value stream step `mq-reconnect-backoff`

### Task 5: Verification

- [x] `go test ./broker/... ./server/... ./domain/...`
- [x] `go test -tags=integration ./integration/...` (requires redis; optional local)
- [x] `broker/redis_backoff_test.go` — QS-01 runtime backoff on dead Redis
- [x] `runAll` config test — task-events split health probe regression
