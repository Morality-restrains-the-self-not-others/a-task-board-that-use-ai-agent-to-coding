# Value Stream: docker-infra conf 碎片同步

> Derived from: `docs/superpowers/specs/2026-06-02-docker-infra-conf-sync-design.md`

## Related Value Streams

- **runall-docker-infra-split**: modification — infra 探活/连接地址改为远程 `172.20.10.7`
- **runall-remote-docker-sync**: extension — conf SSOT 与 remote stack 对齐
- **message-queue-kafka-to-redis**: modification — `redis.host` 来源改为 docker-infra 碎片

## Value Increments

### Increment 1: docker-infra SSOT + sync 碎片（Thin Slice）
**Value:** 改一处 IP，domain-events/task-sse/django 自动一致  
**Scope:** config.yaml、sync manifest、GENERATED 碎片、loader merge

### Increment 2: 测试与 CI
**Value:** 防止 infra host 再次漂移  
**Scope:** pytest、check_conf_sync.sh

### Increment 3: runAll infra 探活
**Value:** runAll 直连远程 Redis/Kafka 健康检查  
**Scope:** runAll/config.yaml infrastructure 组
