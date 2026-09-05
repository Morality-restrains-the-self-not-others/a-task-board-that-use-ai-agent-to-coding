# task2app 价值流 YAML 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让仓库根目录 `value-stream.yaml` 成为 task2app 五条价值流的可视化、可执行编排入口；`planned` 环节随 `view_test` 补齐逐步变为 `active`，并与测试/流程文档保持一致。

**Architecture:** `valueStream`（Go）读取 YAML、校验 `active` 的 pytest 路径、Web UI 展示 `lifecycle` + 运行态；`Saas_project` 按应用模块在 `*/view_test/` 落地文档中的 ViewSet 测试。无 Django 领域层改动——测试文件归属各 app 的接口层（`view_test`），与现有 `tests/` 并存。

**Tech Stack:** Go 1.24（valueStream）、YAML、`pytest` + `saas_project.settings_test`、Django REST ViewSet 测试

**Spec:** [docs/superpowers/specs/2026-05-19-task2app-value-stream-yaml-design.md](../specs/2026-05-19-task2app-value-stream-yaml-design.md)

**Related:** [value-stream-service-design.md](../specs/2026-05-19-value-stream-service-design.md) · [value-stream-service-plan.md](./2026-05-19-value-stream-service-plan.md)

---

## 当前状态（Phase 0 已交付）

| 交付物 | 路径 | 状态 |
|--------|------|------|
| 根配置 | `value-stream.yaml` | 5 流 / 19 planned / 16 active |
| `status` 扩展 | `valueStream/src/config.go`, `status.go`, `runner.go`, `index.html` | 已实现 |
| 设计说明 | `docs/superpowers/specs/2026-05-19-task2app-value-stream-yaml-design.md` | approved |
| 示例指向 | `docs/examples/value-streams.example.yaml` 头部 | 已更新 |

本计划 **Phase 1** 为验收与文档闭合；**Phase 2** 为按价值流迁移 `planned → active`（可分批 PR）。

---

## File Map

| 文件 | 职责 |
|------|------|
| `value-stream.yaml` | 价值流注册（环节、status、fields、test_file） |
| `runAll/config.yaml` | 字段首段服务名白名单 |
| `valueStream/src/*.go` | 解析、校验、执行、API/UI |
| `valueStream/src/config_test.go` | planned 跳过文件校验等 |
| `task2app/Saas_project/accounts/view_test/*.py` | 身份/公司 ViewSet 测试（待建） |
| `task2app/Saas_project/cloud/view_test/*.py` | 云授权/镜像 ViewSet 测试（待建） |
| `task2app/Saas_project/projects/view_test/*.py` | 项目/任务 ViewSet 测试（待建） |
| `task2app/Saas_project/docs/testing/unit-test-cases.md` | 用例与 test_file 对照 |
| `task2app/Saas_project/docs/flows/价值流/.../01_价值流测试集成.wsd` | 价值流测试点图 |

---

## Phase 1 — 验收与文档闭合

### Task 1: 校验 valueStream 与根 YAML 可启动

**Files:**
- Verify: `value-stream.yaml`
- Verify: `runAll/config.yaml`
- Run: `valueStream/build.sh`

- [ ] **Step 1: 编译**

```bash
cd valueStream && ./build.sh
```

Expected: `bin/valueStream` 存在。

- [ ] **Step 2: 配置加载（应无 Config error）**

```bash
cd /path/to/ram-mount
timeout 3 ./valueStream/bin/valueStream --config value-stream.yaml --ui-port :9998 2>&1 || true
```

Expected: 日志含 `listening` 或端口监听；**不得**出现 `Config error` / `test_file not found`（planned 路径除外）。

- [ ] **Step 3: Go 单元测试**

```bash
cd valueStream && go test ./... -race -count=1
```

Expected: `ok valueStream` 与 `ok valueStream/src`。

---

### Task 2: 手工冒烟 — 16 个 active 环节

**Files:**
- Read: `value-stream.yaml`（所有 `status: active` 的 `test_file`）

- [ ] **Step 1: 单文件 pytest（任选一条 active）**

```bash
cd task2app/Saas_project
export DJANGO_SETTINGS_MODULE=saas_project.settings_test
pytest tests/test_login_phone_code.py -v --tb=short
```

Expected: 退出码 `0`（或记录已知失败用例，不得因环境缺失 settings 而崩溃）。

- [ ] **Step 2: UI 整条流 — user-auth**

1. 启动：`./valueStream/bin/valueStream --config value-stream.yaml --ui-port :9998`
2. 打开 http://localhost:9998
3. 对 `user-auth` 点「测试整条流」

Expected: 仅 4 个 active 环节执行；5 个 planned 保持紫色 `planned`、无 pytest 调用；流级灯在 active 全绿时为 `passed`。

- [ ] **Step 3: 单环节 API — planned 应拒绝**

```bash
curl -s -X POST http://localhost:9998/api/test/step \
  -H 'Content-Type: application/json' \
  -d '{"stream":"user-auth","step":"email-register"}' | jq .
```

Expected: HTTP 400，body 含 `planned and cannot be run`。

---

### Task 3: 更新 value-stream-service 设计 spec（status 字段）

**Files:**
- Modify: `docs/superpowers/specs/2026-05-19-value-stream-service-design.md`

在 **YAML Configuration** 的 `steps[]` 表增加一行：

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `status` | no | `active` | `active`：校验 `test_file` 存在且可执行；`planned`：目标用例，跳过文件校验与执行 |

在 **Pass / Fail Semantics** 增加一句：整条流与流级 `failed_steps` **仅统计 `active` 环节**。

- [ ] **Step 1: 编辑 spec 并保存**

- [ ] **Step 2: 自检**

确认 spec 与 `valueStream/README.md`、`2026-05-19-task2app-value-stream-yaml-design.md` 无矛盾。

---

### Task 4: 同步 valueStream README 用法示例

**Files:**
- Modify: `valueStream/README.md`

- [ ] **Step 1: Usage 节增加根配置示例**

```bash
./bin/valueStream --config ../value-stream.yaml --ui-port :9998
```

- [ ] **Step 2: 提交（若用户要求 commit）**

```bash
git add docs/superpowers/specs/2026-05-19-value-stream-service-design.md valueStream/README.md
git commit -m "docs: document value-stream step status active vs planned"
```

---

## Phase 2 — planned → active 迁移（按价值流分批）

**通用迁移步骤（每个 planned 环节重复）：**

1. 在 `task2app/Saas_project/<app>/view_test/` 创建 `unit-test-cases.md` 中对应的 `*_test.py`（参考同目录既有 `tests/test_*.py` 的 fixture/APIClient 模式）。
2. 本地运行：`pytest <app>/view_test/<file>.py -v`（`working_dir` = `Saas_project`）。
3. 修改 `value-stream.yaml`：该 step 的 `status: planned` → `status: active`，确认 `test_file` 路径与文件一致。
4. 重启 valueStream，确认启动无 `test_file not found`；UI 单环节可跑通。
5. 更新 `unit-test-cases.md` 与 `01_价值流测试集成.wsd` 中该测试点状态（若团队有「已自动化」标记约定）。

**pytest 路径说明：** 根 YAML 的 `test_file` 相对 `runner.working_dir`（`task2app/Saas_project`）。若 `pytest.ini` 仅 `testpaths = tests`，对 `view_test` 仍可用**显式文件路径**调用（valueStream 即如此），无需改 `pytest.ini`。

---

### Task 5: user-auth — 5 个 planned 环节

**Files:**
- Create: `task2app/Saas_project/accounts/view_test/UserViewSet_email_register_test.py`
- Create: `task2app/Saas_project/accounts/view_test/UserViewSet_activate_test.py`
- Create: `task2app/Saas_project/accounts/view_test/UserViewSet_login_test.py`
- Create: `task2app/Saas_project/accounts/view_test/UserViewSet_reset_password_test.py`
- Create: `task2app/Saas_project/accounts/view_test/UserViewSet_resend_activation_email_test.py`
- Modify: `value-stream.yaml`（`user-auth` 流 5 处 status）
- Test: `task2app/Saas_project/docs/testing/unit-test-cases.md` §1

- [ ] **Step 1: 建目录**

```bash
mkdir -p task2app/Saas_project/accounts/view_test
touch task2app/Saas_project/accounts/view_test/__init__.py
```

- [ ] **Step 2: 实现 `UserViewSet_email_register_test.py`（最小：1 个 happy path）**

参考 `docs/testing/unit-test-cases.md` §1.1 表格「正常注册」；使用 `APIClient` + `pytest.mark.django_db`。

- [ ] **Step 3: 运行并绿**

```bash
cd task2app/Saas_project && pytest accounts/view_test/UserViewSet_email_register_test.py -v
```

- [ ] **Step 4: 将 `email-register` step 改为 active**

`value-stream.yaml` 中 `name: email-register` 的 `status: active`。

- [ ] **Step 5: 对其余 4 个 planned 重复 Step 2–4**（activate、login、reset-password、resend-activation）

- [ ] **Step 6: valueStream 验收**

```bash
./valueStream/bin/valueStream --config value-stream.yaml
# UI：user-auth 整条流应跑 9 个 active（原 4 + 新 5）
```

- [ ] **Step 7: Commit（一批或五个小 commit，按仓库习惯）**

```bash
git add task2app/Saas_project/accounts/view_test/ value-stream.yaml
git commit -m "test(accounts): add user-auth view_test and activate value-stream steps"
```

---

### Task 6: company-management — 3 个 planned

**Files:**
- Create: `accounts/view_test/CompanyViewSet_test.py`
- Create: `accounts/view_test/CompanyMemberViewSet_test.py`
- Create: `accounts/view_test/UserViewSet_test.py`
- Modify: `value-stream.yaml`（`company-management`）

- [ ] **Step 1–3:** 同 Task 5 模式，对照 `unit-test-cases.md` §2。

- [ ] **Step 4:** 三条 step 均改为 `active` 后跑 valueStream 整条 `company-management`（共 5 active）。

---

### Task 7: cloud-integration — 4 个 planned

**Files:**
- Create: `cloud/view_test/CloudPlatformAuthorizationViewSet_test.py`
- Create: `cloud/view_test/CloudServerImageViewSet_test.py`
- Create: `cloud/view_test/CloudServerImageViewSet_get_images_test.py`
- Create: `cloud/tests/test_cloud_compute.py`（若不存在；YAML 已指向此路径）
- Modify: `value-stream.yaml`（`cloud-integration`）

- [ ] **Step 1:** `mkdir -p cloud/view_test && touch cloud/view_test/__init__.py`

- [ ] **Step 2–4:** 按 §3 实现并 pytest 绿。

- [ ] **Step 5:** 4 个 planned → active；整条 `cloud-integration` 应为 9 active。

---

### Task 8: project-workspace — 4 个 planned

**Files:**
- Create: `projects/view_test/WorkspaceViewSet_test.py`
- Create: `projects/view_test/ProjectViewSet_test.py`
- Create: `projects/view_test/DeliverableSystemViewSet_test.py`
- Create: `projects/view_test/WorkspaceViewSet_switch_workspace_test.py`
- Modify: `value-stream.yaml`（`project-workspace`）

- [ ] **Step 1:** `mkdir -p projects/view_test && touch projects/view_test/__init__.py`

- [ ] **Step 2–4:** 对照 §4；4 planned → active（流内共 6 active）。

---

### Task 9: task-management — 3 个 planned

**Files:**
- Create: `projects/view_test/TodoViewSet_test.py`
- Create: `projects/view_test/TodoViewSet_manage_status_test.py`
- Create: `projects/view_test/TodoViewSet_manage_status_fixed_test.py`
- Modify: `value-stream.yaml`（`task-management`）

- [ ] **Step 1–3:** 对照 §5；3 planned → active（流内共 6 active）。

- [ ] **Step 4: 全仓价值流终验**

```bash
cd valueStream && go test ./... -race
./bin/valueStream --config ../value-stream.yaml
# 依次对 5 条流跑「测试整条流」，记录失败环节列表
```

Expected: 无 planned 环节；共 35 个 step 均为 active；启动时 35 个 `test_file` 均通过存在性校验。

---

### Task 10: 文档与图示同步

**Files:**
- Modify: `task2app/Saas_project/docs/testing/unit-test-cases.md`
- Modify: `task2app/Saas_project/docs/testing/unit-test-distribution.md`
- Modify: `task2app/Saas_project/docs/flows/价值流/01_项目管理价值流/业务架构/01_价值流测试集成/01_价值流测试集成.wsd`
- Modify: `docs/superpowers/specs/2026-05-19-task2app-value-stream-yaml-design.md`（Status → implemented）

- [ ] **Step 1:** 在 `unit-test-distribution.md` 注明「编排入口：仓库根 `value-stream.yaml`」。

- [ ] **Step 2:** 更新 wsd 注释中的测试文件路径与 valueStream 一致。

- [ ] **Step 3:** 将设计 spec 的 **Status** 改为 `implemented` 并注明日期。

---

## Spec 覆盖自检

| Spec 要求 | 计划任务 |
|-----------|----------|
| 根目录 `value-stream.yaml` 五条流 | Phase 0 已交付；Task 1 验收 |
| `status: active \| planned` | Phase 0；Task 3 写入 service spec |
| 16 active 可执行 | Task 2 |
| 19 planned 可迁移 | Task 5–9 |
| runAll 字段白名单 | Task 1（`runAll/config.yaml`） |
| 维护约定（文档同步） | Task 10 |
| UI lifecycle + planned 不可单跑 | Task 2 Step 3 |

## 占位符扫描

- 无 TBD/TODO 步骤。
- Phase 2 各 `view_test` 实现需对照 `unit-test-cases.md` 具体表格填写断言（实施时按表复制输入/预期）。

---

## 建议执行顺序

1. **Phase 1（Task 1–4）** — 半天内可完成，确认已交付代码可合并。
2. **Phase 2** — 按依赖流顺序 Task 5 → 6 → 7 → 8 → 9；每流可独立 PR。
3. **Task 10** — 可与最后一个 view_test PR 同批或紧随其后。

**DDD 说明：** valueStream 为运维编排工具，不适用分层领域模型。`Saas_project` 新增测试仅放在各 app 的 `view_test/`，不新增 domain 实体或 repository。
