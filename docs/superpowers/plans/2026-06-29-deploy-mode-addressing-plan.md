# 实施计划: DEPLOY_MODE 寻址模式切换

> 上游:
> - 设计: `.claude/skills/1-brainstorming-设计文档/design.md`
> - 价值流: `docs/superpowers/plans/2026-06-29-deploy-mode-addressing-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-29-deploy-mode-addressing-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-29-deploy-mode-addressing-ddd.md`

## 任务清单

### Increment 1: Python 配置解析核心 (Thin Slice)

- [ ] **T1.1** — 新增 `conf/base.yaml` 补充 `gateway` 寻址键
  - 文件: `conf/base.yaml`
  - 内容: `subdomains.gateway: ${baseDomain}` + `local.gateway: ${GATEWAY_ADDR:-183.250.1.132:18081}`
  - 验证: YAML 语法有效 `python3 -c "import yaml; yaml.safe_load(open('conf/base.yaml'))"`

- [ ] **T1.2** — 新增 `DeployMode` 值对象
  - 文件: `runAll/scripts/domain/value_objects/deploy_mode.py`
  - 契约: 接受 `"domain"` / `"local"` 字符串，其他值抛 `ValueError`
  - 测试: `runAll/scripts/tests/test_deploy_mode.py`

- [ ] **T1.3** — 新增 `AddressingScheme` 值对象
  - 文件: `runAll/scripts/domain/value_objects/addressing_scheme.py`
  - 契约: `resolve(key)` 返回地址; `resolve_template(value)` 替换占位符
  - 约束: domain 模式地址不含端口，local 模式地址为 `host:port`
  - 测试: `runAll/scripts/tests/test_addressing_scheme.py`

- [ ] **T1.4** — 新增 `BaseYAMLLoader` 领域服务
  - 文件: `runAll/scripts/domain/services/base_yaml_loader.py`
  - 契约: `load(path) -> AddressingScheme`
  - 规则: 读 base.yaml → 解析 mode → 展开对应段 → 环境变量覆盖
  - 测试: `runAll/scripts/tests/test_base_yaml_loader.py`

- [ ] **T1.5** — 新增 `TemplateResolver` 领域服务
  - 文件: `runAll/scripts/domain/services/template_resolver.py`
  - 契约: `resolve(config: dict, scheme: AddressingScheme) -> dict`
  - 规则: 递归替换 `${subdomains.xxx}` / `${baseDomain}`; 纯函数; 返回新 dict
  - 测试: `runAll/scripts/tests/test_template_resolver.py`

- [ ] **T1.6** — 修改 `conf_loader.py` 集成模板解析
  - 文件: `runAll/scripts/conf_loader.py`
  - 变更: `load_app_config()` 末尾调用 `TemplateResolver.resolve()`
  - 验证: 现有 `conf-read.py` 输出与当前一致（DEPLOY_MODE=local）

- [ ] **T1.7** — 修改 `conf-read.py` 注入 `_addressing`
  - 文件: `runAll/scripts/conf-read.py`
  - 变更: `build_runtime_snapshot()` 追加 `_addressing` 键
  - 验证: `python3 runAll/scripts/conf-read.py snapshot-json | python3 -c "import sys,json; d=json.load(sys.stdin); assert '_addressing' in d"`

### Increment 2: 网关路由生成适配

- [ ] **T2.1** — 修改 `routes-to-apisix.py` 读 base.yaml
  - 文件: `taskGateway/scripts/routes-to-apisix.py`
  - 变更: `_upstream_host()` 调用 `load_base_yaml()`，用 addressing 覆盖 hardcoded host
  - 规则: 解析成功用 resolved host; 解析失败阻止启动（不吞错）
  - 验证: `python3 taskGateway/scripts/routes-to-apisix.py --check` 通过

- [ ] **T2.2** — 修改 `conf/gateway/task-gateway/config.yaml` — `publicBase`
  - 文件: `conf/gateway/task-gateway/config.yaml`
  - 变更: `publicBase: http://183.250.1.132:18081` → `publicBase` 通过 base.yaml 解析
  - 验证: `DEPLOY_MODE=local` `conf-read.py taskGateway publicBase` 输出 `http://183.250.1.132:18081`

- [ ] **T2.3** — 验证生成的 apisix.yaml 一致性
  - 命令: `DEPLOY_MODE=local python3 taskGateway/scripts/routes-to-apisix.py && diff taskGateway/apisix/apisix.yaml <(git show HEAD:taskGateway/apisix/apisix.yaml)`
  - 预期: 无差异

### Increment 3: 服务配置迁移

- [ ] **T3.1** — 迁移 `conf/auth/task-auth/config.yaml` OIDC 字段
  - 字段: `oidc.issuer`, `oidc.bootstrapRedirectUri`, `oidc.gitServicePublicBase`
  - 变更: 硬编码 `183.250.1.132:*` → `${subdomains.gateway}` / `${subdomains.gitlab}` 模板
  - 验证: `DEPLOY_MODE=local` `conf-read.py taskAuth oidc.issuer` 输出 `http://183.250.1.132:18081`

- [ ] **T3.2** — 迁移 `conf/frontend/vue/config.yaml` 对外字段
  - 字段: `publicBaseUrl`, `apiBaseUrl`
  - 变更: 硬编码 IP → `${subdomains.www}` / `${subdomains.gateway}` 模板
  - 验证: `DEPLOY_MODE=local` 解析值与当前一致

- [ ] **T3.3** — 迁移 `conf/frontend/vue/git-service.yaml` 对外字段
  - 字段: `publicUrl`
  - 变更: `http://183.250.1.132:8012` → `http://${subdomains.gitlab}`
  - 验证: `DEPLOY_MODE=local` 解析值与当前一致

- [ ] **T3.4** — 迁移 `conf/infra/git-service/config.yaml` 对外字段
  - 字段: `allowedHost`, `publicUrl`
  - 变更: 硬编码 IP → 模板变量
  - 验证: `DEPLOY_MODE=local` 解析值与当前一致

- [ ] **T3.5** — 迁移 `conf/gateway/task-gateway/config.yaml` CORS 字段
  - 字段: `cors.allowedOrigins`
  - 变更: 硬编码 `183.250.1.132:*` → `${subdomains.www}` 模板
  - 验证: `DEPLOY_MODE=local` 解析值与当前一致

### 端到端验证

- [ ] **T4.1** — DEPLOY_MODE=local 全量回归
  - 命令: `DEPLOY_MODE=local python3 runAll/scripts/conf-read.py snapshot-json > /tmp/snapshot_after.json && diff <(python3 runAll/scripts/conf-read.py snapshot-json | python3 -c "import sys,json; d=json.load(sys.stdin); d.pop('_addressing',None); print(json.dumps(d,sort_keys=True))") <(git stash && DEPLOY_MODE=local python3 runAll/scripts/conf-read.py snapshot-json | python3 -c "import sys,json; d=json.load(sys.stdin); d.pop('_addressing',None); print(json.dumps(d,sort_keys=True)); git stash pop")`
  - 预期: 除 `_addressing` 外无差异

- [ ] **T4.2** — DEPLOY_MODE=domain 快照验证
  - 命令: `DEPLOY_MODE=domain python3 runAll/scripts/conf-read.py snapshot-json | python3 -c "import sys,json; d=json.load(sys.stdin); a=d['_addressing']; assert a['mode']=='domain'; print(json.dumps(a,indent=2))"`
  - 预期: mode=domain, addresses 为域名格式（无端口）

- [ ] **T4.3** — CI 检查通过
  - 命令: `bash runAll/scripts/conf-sync-all.sh && bash scripts/ci/check_conf_sync.sh`

## 任务依赖

```
T1.1 ──┬── T1.2 ── T1.3 ── T1.4 ── T1.5 ── T1.6 ── T1.7
       │
       └── T2.1 ── T2.2 ── T2.3
                         │
       T3.1 ── T3.2 ── T3.3 ── T3.4 ── T3.5
                                           │
                                     T4.1 ── T4.2 ── T4.3
```

T1.x (Increment 1) 是 T2.x 和 T3.x 的前置。T3.x 之间无强制顺序。T4.x 在所有实现完成后执行。

## 预估影响

| 指标 | 数值 |
|------|------|
| 新增文件 | ~8 (4 domain + 4 test) |
| 修改文件 | ~9 (base.yaml + 3 conf_lib/* + 1 routes-to-apisix + 4 config.yaml) |
| 删除文件 | 0 |
| 测试新增 | ~4 test files |
