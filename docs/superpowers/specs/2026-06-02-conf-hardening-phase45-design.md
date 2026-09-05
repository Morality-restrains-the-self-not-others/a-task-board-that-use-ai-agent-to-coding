# conf/ 硬化设计（Phase 4.5 — 审查扫尾）

> 日期：2026-06-02  
> 状态：**已实施**（Phase 4.5a–e，2026-06-02 auto-flow）  
> 前置：`2026-06-02-port-config-split-monorepo-conf-design.md`（Phase 1–3 已实施）  
> 触发：`/8-review` 识别的中风险与低风险项

---

## 1. 背景与动机

Phase 1–3 已完成：`port_config.json` 已删、`<monorepo>/conf/<app>/` 为权威源、sync 碎片与多数消费者已切 YAML。

**仍存在的差距：**

| 差距 | 影响 |
|------|------|
| Django 仍经 `assemble_legacy_port_config()` / `read_port_config()` 拼装「旧 JSON 形状」 | 与设计 §1「直读、删聚合层」字面不一致；`load_django_fragments()` 通配 `conf/core/django/*.yaml` 有误读风险 |
| 无 CI 校验 conf 与 sync 一致 | 手改权威 YAML 或碎片可静默漂移 |
| `write_port_config()` 改端口后不跑 sync | 碎片与权威 `config.yaml` 短暂不一致 |
| 敏感凭据仍在 `conf/core/django/config.yaml` 明文 | 迁移前即存在；本期仅文档化分层，不迁出明文 |
| `task2app-wt-relay-stop` 仍读 JSON | 主树 PR 内一并改为 conf 策略 |
| `scripts/conf-read.py` 未实现 | shell 仍散落 ad-hoc Python |
| `go_run_container.configFilePath()` 死代码 | 测试/env 契约不清晰 |

本设计 **不重复** Phase 1–3 迁移，只闭合审查项。

---

## 2. 目标与完成标准

| # | 完成标准 | 可验证 |
|---|----------|--------|
| G1 | 全仓运行时不再依赖 `assemble_legacy_port_config()`；**删除** `scripts/conf_emit_json.py` | `rg assemble_legacy_port_config` / `conf_emit_json` 无运行时引用 |
| G1b | Playwright / Vite **直读** `conf/<app>/*.yaml`（或共享 `conf-read` / 小型 TS loader），不 exec 聚合 JSON | 无 `loadPortConfig.mjs` 调 `conf_emit_json.py` |
| G2 | CI：`conf-sync-all.sh` 后 **`conf/**` 零 diff** | `scripts/ci/check_conf_sync.sh` 纳入合规链 |
| G3 | `write_port_config()` 写盘后自动 `conf-sync`（django/vue 及相关 manifest） | 单测 + G2 兜底 |
| G4 | 密钥：**仅** `config.local.yaml` + `config.example.yaml` + README 文档；**不**从 `config.yaml` 迁出明文 | 文档与 example 提交；`config.local.yaml` gitignore 说明 |
| G5 | `scripts/conf-read.py <app> <dot.path>` | shell/文档引用收敛 |
| G6 | **主树 PR** 直接更新 `task2app-wt-relay-stop/`：无 `port_config.json` 运行时读，与主 `task2app/` 同策略 | `rg port_config.json task2app-wt-relay-stop` 无 `.py/.js/.sh` 读路径 |

**非目标（本期）：**

- 不改各服务业务端口号语义
- 不实现 `runAll.yaml` 从 `conf/` 自动生成
- 不合并 `Saas_Ai_Provider/conf/conf.yaml` 业务配置
- **D2=A**：不从 `config.yaml` 物理迁出已有明文密钥、不轮换密钥

---

## 3. 已确认方案

### 3.1 去聚合层（G1 + G1b）— **D1=B**

**原则：** 各模块只读自己需要的 `conf/<app>/`；禁止「整棵 port_config dict」作为隐式总线。

| 消费者 | 目标 |
|--------|------|
| `settings_manager` / `settings.py` | `load_app_config('django')` 等；`gitOauth` → `load_git_oauth_catalog()` |
| `ai_provider_config` | `load_app_config('ai-provider')` |
| `health/checks` | `load_app_config('domain-events')`（redis 段） |
| `daydaymoney_nginx_conf.py` | `load_app_config('django')` + `load_app_config('vue')` |
| `port_config.py` | 窄接口或删除；`read_port_config` 移除或 stub 报错 |
| Playwright | `helpers/loadConfYaml.mjs`（或按 app 读 yaml）；**删除** `conf_emit_json` 依赖 |
| Vite（front_project、Saas_Ai_Provider） | `js-yaml` / 小脚本读 `conf/vue`、`conf/core/django` 所需键，或 `conf-read.py` |
| `load_django_fragments()` | **删除** |

**删除制品：**

- `scripts/conf_emit_json.py`
- `task2app/playwright/helpers/loadPortConfig.mjs`（替换为 YAML loader）

### 3.2 CI（G2）— **D4=B**

```bash
# scripts/ci/check_conf_sync.sh
./scripts/conf-sync-all.sh
git diff --exit-code -- conf/
```

- 检查范围：**整个 `conf/`**（含权威 `config.yaml` 与 GENERATED 碎片）
- 含义：仓库内 `conf/` 必须与 sync 管线输出完全一致；手改权威 YAML 若未更新 manifest 源逻辑，CI 会失败（迫使先改 owner `config.yaml` 再 sync）
- 挂到 `check_ddd_bdd_compliance` 链或独立 job

### 3.3 write_port_config + sync（G3）

`write_port_config()` 末尾调用 `conf-sync.py django` / `vue`（及受影响 app）；失败 `warnings.warn`；CI 用 G2 兜底。

### 3.4 密钥分层（G4）— **D2=A**

| 层 | 内容 | git |
|----|------|-----|
| `config.yaml` | 现状保持（含既有明文密钥） | 提交 |
| `config.local.yaml` | 文档约定：未来/local 覆盖密钥 | **gitignore**（新增规则 + example） |
| `config.example.yaml` | 键名模板、占位符 | 提交 |

**本期不做：** 从 `config.yaml` 抽离 secret、密钥轮换。

### 3.5 conf-read CLI（G5）

```text
scripts/conf-read.py django port
scripts/conf-read.py domain-events transport
scripts/conf-read.py git-oauth --provider <file-stem> target.client_id
```

供 shell、Vite 预加载脚本复用。

### 3.6 worktree（G6）— **D3=B**

- **主树 PR 内**直接修改 `task2app-wt-relay-stop/`：与 `task2app/` 相同的 conf 加载、Playwright、vite、run.sh 补丁
- 删除 `task2app-wt-relay-stop/conf/port_config.json`（若仍存在）
- 不单独写「仅合并指南」文档为主交付物（可在 PR 描述中附 rebase 提示）

### 3.7 go_run_container

删除未使用的 `configFilePath()`，或接通 `MOCK_RUN_CONTAINER_CONFIG_PATH` 单文件 YAML 覆盖。

---

## 4. 价值流影响

| 问题 | 评估 |
|------|------|
| 受影响 stream | `domain-events-consumer-split`、`message-queue-kafka-to-redis`、`git-site-oauth-*` |
| 字段 description | `port_config.*` → `conf/<app>/config.yaml` 或 `conf/domain-events/<event>/config.yaml` |
| 测试 | `test_port_config_merge` → `test_conf_loader`；Playwright 改 YAML fixture；新增 conf-sync CI 测试 |
| 新 stream | 不需要 |

---

## 5. 领域概念清单（供 `/5-ddd`）

| 概念 | 说明 |
|------|------|
| **Bounded Context** | Platform Runtime Config |
| **Entity** | `AppConfig` |
| **Value Object** | `ConfigFragment`、`SyncManifestEntry` |
| **Aggregate** | `DomainEventBundle` |

---

## 6. 实施分期（按已批决策重排）

| Phase | 内容 |
|-------|------|
| **4.5a** | G2（`conf/**` 零 diff CI）+ G3 write/sync + G5 conf-read + go 死代码 |
| **4.5b** | G1 Saas 直读 + 删 `assemble_legacy` / `load_django_fragments` |
| **4.5c** | G1b 删 `conf_emit_json`；Vite/Playwright 直读 YAML |
| **4.5d** | G4 文档 + `config.example.yaml`；G6 **主树改** `task2app-wt-relay-stop` |
| **4.5e** | `value-stream.yaml` 描述更新；全量验证 |

---

## 7. 已锁定决策（2026-06-02）

| # | 选择 | 含义 |
|---|------|------|
| **D1** | **B** | 含删 `conf_emit_json` + Playwright/Vite 直读 YAML |
| **D2** | **A** | 只做 `local` + `example` + 文档；不迁出 `config.yaml` 明文密钥 |
| **D3** | **B** | 主树 PR **直接改** `task2app-wt-relay-stop` |
| **D4** | **B** | CI：`conf-sync-all.sh` 后 **整个 `conf/`** 零 diff |

---

## 8. Spec 自检

- [x] 与 Phase 1–3 设计无冲突  
- [x] 价值流影响已列  
- [x] 用户确认 D1–D4  

---

## 9. 下一步（流程）

→ `/6-plans-实施计划`（勾选 4.5a–e）→ `/7-build-构建`（按分期 TDD 实施）。

可选前置：`/3-value-stream-价值流`（批量改 `port_config` 字段描述）。**D3=B** 时不必单独 `/2-worktrees` 隔离 wt-relay-stop（与主 PR 同批交付）。
