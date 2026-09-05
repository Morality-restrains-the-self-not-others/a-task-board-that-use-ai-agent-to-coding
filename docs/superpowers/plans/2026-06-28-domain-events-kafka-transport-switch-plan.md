# Implementation Plan: Domain Events Transport Redis→Kafka 切换

> Inputs:
> - Design: `docs/design/consumer-groups-kafka-transport-switch.md`
> - Value Stream: `docs/superpowers/plans/2026-06-28-domain-events-kafka-transport-switch-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-28-domain-events-kafka-transport-switch-nfr.md`
> - DDD: `docs/superpowers/plans/2026-06-28-domain-events-kafka-transport-switch-ddd.md`

## Summary

3 file changes: 2 config files + 1 test assertion. Code already supports both transports.

## Task List

### Task 1: Update transport config — `conf/domain-events/config.yaml`

- [ ] Change `transport: redis` → `transport: kafka`
- **File**: `conf/domain-events/config.yaml`
- **Verify**: `grep 'transport:' conf/domain-events/config.yaml` returns `transport: kafka`

### Task 2: Add bootstrapServers — `conf/infra/docker-infra/config.yaml`

- [ ] Add `bootstrapServers: localhost:9093` to `kafka:` block
- **File**: `conf/infra/docker-infra/config.yaml`
- **Verify**: `grep bootstrapServers conf/infra/docker-infra/config.yaml` returns the value

### Task 3: Update test assertion — `tests/test_domain_events_port_config.py`

- [ ] Line 12: `assert block.get("transport") == "redis"` → `assert block.get("transport") == "kafka"`
- **File**: `task2app/Saas_project/tests/test_domain_events_port_config.py`
- **Verify**: `cd task2app/Saas_project && source activate_env.sh && python -m pytest tests/test_domain_events_port_config.py -x`

### Task 4: Run domain events transport test

- [ ] Execute `test_domain_events_port_config.py` and confirm all assertions pass
- **Command**: `cd task2app/Saas_project && source activate_env.sh && python -m pytest tests/test_domain_events_port_config.py -v`
- **Expected**: 1 passed

### Task 5: Restart taskEvents consumers and verify Kafka UI

- [ ] Restart taskEvents via runAll: stop → start (consumers re-read config on startup)
- [ ] Verify `http://183.250.1.132:18080/ui/clusters/local/consumer-groups` shows consumer groups
- **Expected**: 6 consumer groups visible with member count ≥ 1

### Task 6: Run integration test — registration chain

- [ ] Execute `taskEvents/integration/registration_chain_test.go` to verify end-to-end event flow through Kafka
- **Command**: `cd taskEvents && go test ./integration/ -run TestRegistrationChain -v`
- **Expected**: PASS — user created → company created chain works via Kafka

## Task Dependencies

```
Task 1 ──→ Task 2 ──→ Task 3 ──→ Task 4 ──→ Task 5 ──→ Task 6
```

## Rollback Plan

If Task 4 or Task 5 fails:
```bash
# Revert config
sed -i 's/transport: kafka/transport: redis/' conf/domain-events/config.yaml
# Restart consumers
# Rerun test
```
