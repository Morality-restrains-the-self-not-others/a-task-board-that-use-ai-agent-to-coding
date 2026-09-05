# 实施计划: docker-infra conf 碎片同步

## Task 1: docker-infra SSOT
- [ ] 重写 `conf/docker-infra/config.yaml`（host=172.20.10.7）
- [ ] 从 domain-events/task-sse 移除手写 redis.host / kafka.bootstrap

## Task 2: sync manifests
- [ ] 更新 `conf/domain-events/sync.manifest.yaml`
- [ ] 新建 `conf/task-sse/sync.manifest.yaml` + `sync.sh`
- [ ] 更新 `conf/core/django/sync.manifest.yaml`
- [ ] 运行 `./scripts/conf-sync-all.sh`

## Task 3: loaders
- [ ] `conf_loader.py`: merge `docker-infra.yaml` 碎片
- [ ] `confload/load.go`: merge 碎片
- [ ] `taskEvents/config/conf_yaml.go`: merge 碎片
- [ ] `taskSSE/src/config.mjs`: merge 碎片

## Task 4: 测试
- [ ] 扩展 `test_domain_events_port_config.py` 断言 redis.host=172.20.10.7
- [ ] `./scripts/ci/check_conf_sync.sh`

## Task 5: runAll
- [ ] infrastructure 组探活改为 172.20.10.7
