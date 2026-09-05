# 实施计划：大流量开发期护栏

设计：`docs/superpowers/specs/2026-07-15-high-traffic-dev-hardening-design.md`

## NOW

- [x] N1 APISIX `globalPerMinute` → global_rules `limit-req`
- [x] N2 Go SQLite MaxOpenConns + busy_timeout（project/task/cloud/ai-comment/credential）
- [x] N3 Django busy_timeout 30000 + 已有连接补设
- [x] N4 taskSSE `maxConnections`（默认 500）
- [x] N5 AI instruct chunk SSE 批处理
- [x] N6 forward-auth 2s 缓存
- [x] N7 relay lifecycle 锁超时 503
- [x] N8 Intent / 测试意图 / 设计文档

## PRE-PROD（未做）

- [ ] P1 Postgres 迁移
- [ ] P2 Redis session
- [ ] P3 多 relay 分片
- [ ] P4 熔断 + 云 API 超时统一
- [ ] P5 Redis/Kafka HA
- [ ] P6 django-legacy 继续迁 Go
