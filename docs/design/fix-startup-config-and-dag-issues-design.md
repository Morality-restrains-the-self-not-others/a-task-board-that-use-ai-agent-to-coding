# Design: Fix runAll Startup, Config Path, and DAG Issues

**Status**: draft
**Created**: 2026-06-23
**Type**: bugfix

## Problem Summary

Production logs show four interrelated issues:

```
[10:35:14] [runAll] default state: not started — no process launched yet
[10:35:55] [valueStream] Config error: runall_config: read runall config: open /tmp/ram-work/conf/conf/runAll.yaml: no such file or directory
[10:39:50] [runAll] start: DAG level 1/1 — starts now with 20 peer(s) in parallel
[10:39:50] [runAll] start requested (previous status=failed)
```

## Root Cause Analysis

### Issue 1: `conf/conf/runAll.yaml` — Double `conf/` Path (PRIMARY BUG)

**File**: `conf/value-stream.yaml` line 2
**Current**: `runall_config: conf/runAll.yaml`

**Chain of events**:
1. `productionValueStreamConfigPath()` returns `<repoRoot>/conf/value-stream.yaml`
2. `LoadConfig()` sets `ConfigDir = filepath.Dir(absPath)` → `/tmp/ram-work/conf/`
3. `validate()` line 228: `filepath.Join(c.ConfigDir, c.RunallConfig)` → `/tmp/ram-work/conf/conf/runAll.yaml`

**Root cause**: `conf/value-stream.yaml` was written when the config file was at the repo root. After `repo_root.go` was updated to point to `conf/value-stream.yaml`, the `runall_config` value was not updated to be relative to the new `ConfigDir`.

The README already documents the correct convention:
> When placing value-stream.yaml in conf/, use `runall_config: runAll.yaml`

**Fix**: Change `conf/value-stream.yaml` line 2 to `runall_config: runAll.yaml`.

### Issue 2: DAG Single-Level — All 20 Services in DAG Level 1 (DESIGN BUG)

The `BuildDAG()` function in `runAll/src/dag.go` uses Kahn's algorithm correctly. However, the current runAll.yaml declares `depends_on` relationships, yet the log shows `DAG level 1/1 — starts now with 20 peer(s) in parallel`.

**Analysis of dependency graph** (from `conf/runAll.yaml`):

```
Level 0 (no deps): docker-redis, docker-kafka, docker-portainer, task-auth, task-bill, task-sse
Level 1 (depend on L0): ai-monitor→docker-redis, git-service→docker-redis
Level 2 (depend on L1+): git-oauth→git-service, saas-backend→task-auth+git-oauth+docker-redis+task-sse
Level 3 (depend on L2+): ai-provider→saas-backend, task-gateway→task-auth+git-oauth+saas-backend, 
                          task-agent-support→saas-backend, task-ai-endpoint→saas-backend,
                          task-container-gateway→saas-backend, taskFE→task-gateway+saas-backend
Level 4 (depend on L3): promtail-local→ai-monitor
```

This should produce **5 DAG levels**, not 1. The fact that all 20 services land in level 1 suggests one of:
- The dependency graph is not being computed across groups (group isolation)
- The `depends_on` references are not resolving correctly (cross-group service name mismatch)
- This is a "start all groups in parallel" rather than "start all services with full dependency ordering"

**Fix needed**: Verify the DAG builder receives cross-group dependencies. If the current architecture builds DAGs per-group for isolation, document this as intended behavior. If cross-group dependencies should be respected, fix the orchestrator to merge groups before DAG computation.

### Issue 3: runAll Status Messages to stderr (COSMETIC)

`[runAll] default state: not started — no process launched yet` and `[runAll] start requested (previous status=failed)` are sent to stderr via `log.Printf`. These are informational/status messages, not errors. 

**Fix**: Route runAll lifecycle status messages to stdout or to the structured log system instead of stderr. The `previous status=failed` message is useful for debugging but should include context about *why* the previous run failed.

## Design

### Fix 1: Correct `runall_config` Path

**Change**: `conf/value-stream.yaml` line 2

```yaml
# Before
runall_config: conf/runAll.yaml

# After
runall_config: runAll.yaml
```

This is a one-line fix. The `ConfigDir` is already `conf/`, so the relative path `runAll.yaml` resolves correctly.

### Fix 2: DAG — Verify Cross-Group Dependency Resolution

**Investigation needed** in `runAll/src/dag.go` and the runner:
1. Does `BuildDAG` receive all services across all groups, or only per-group?
2. Do cross-group `depends_on` references resolve correctly?

If the DAG is intentionally per-group (group-level parallelism), then the log message should clarify:
```
[runAll] start: starting group "infrastructure" DAG level 1/3 — 6 peers
[runAll] start: starting group "platform" DAG level 1/4 — 11 peers
```

If cross-group dependencies should merge:
- Merge all groups' services into a single service list before `BuildDAG()`
- Ensure `depends_on` names are unique across groups (they already appear to be)

### Fix 3: runAll Stderr Hygiene

- Move informational lifecycle messages (`default state: not started`, `start requested`) to stdout
- Keep actual errors on stderr
- Add context to `previous status=failed`: include timestamp and reason for the previous failure

## Affected Files

| File | Change |
|------|--------|
| `conf/value-stream.yaml` | Line 2: `runall_config: runAll.yaml` |
| `runAll/src/dag.go` | Investigate cross-group DAG resolution |
| `runAll/src/runner.go` | Investigate how groups are passed to DAG builder |
| `runAll/src/main.go` | Move status messages from stderr to stdout |
| `valueStream/src/config.go` | Consider adding a warning when resolved runall_config path doesn't exist, with suggested fix |

## Value Stream Impact

This fix touches the following existing value streams:

- **`value-stream-config-governance`** (系统管理与策略): The `production-config-validation` step validates `conf/value-stream.yaml` via `LoadConfig`. The current `conf/conf/` bug is caught by this validation at startup (the error message in the logs IS the validation). Fixing the path makes validation pass.

- **`runall-cascade-lifecycle`** (云平台与资源): If the DAG fix changes startup ordering, cascade lifecycle tests need review.

## Domain Concept Inventory

- **Bounded Context**: 平台编排 (Platform Orchestration) — runAll service lifecycle management
- **Key Entities**: `Service` (with DependsOn, HealthCheck), `Config` (value-stream with runall_config reference)
- **Candidate Aggregates**: `ServiceGroup` as a consistency boundary for intra-group startup ordering
- **Domain Events**: `ServiceStarted`, `ServiceFailed`, `ConfigValidationFailed`
