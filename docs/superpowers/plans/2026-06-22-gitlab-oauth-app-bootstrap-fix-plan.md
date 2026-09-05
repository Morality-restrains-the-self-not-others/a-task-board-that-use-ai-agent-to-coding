# 实施计划: GitLab OAuth Application 启动期自愈创建

> 输入:
> - 设计文档: `docs/superpowers/specs/2026-06-22-gitlab-oauth-app-bootstrap-fix-design.md`
> - 价值流: `docs/superpowers/plans/2026-06-22-gitlab-oauth-app-bootstrap-fix-value-stream.md`
> - NFR: `docs/superpowers/plans/2026-06-22-gitlab-oauth-app-bootstrap-fix-nfr-clarification.md`
> - DDD: `docs/superpowers/plans/2026-06-22-gitlab-oauth-app-bootstrap-fix-ddd.md`
>
> **规模:** 小型 bug 修复 — 单文件，约 15 行 net-new Ruby

## 任务清单

### 阶段 1: 测试先行（红）

- [ ] **Task 1.1** — 创建集成测试脚本
  - **文件:** `gitService/scripts/test_sync_oauth_app.sh`（新建）
  - **内容:** 验证 find-or-create 行为的独立测试脚本：
    1. 容器未运行时跳过
    2. 调用 `sync_local_oauth_app_scopes.sh`
    3. 通过 `gitlab-rails runner "puts Doorkeeper::Application.find_by(uid: '...').scopes"` 验证 Application 存在
    4. 第二次调用验证幂等性（相同 uid，Application 数量不变）
  - **命令:** `bash gitService/scripts/test_sync_oauth_app.sh`
  - **依赖:** 无

- [ ] **Task 1.2** — 运行测试（预期失败—红）
  - **命令:** `bash gitService/scripts/test_sync_oauth_app.sh`
  - **预期结果:** 在全新容器上：Application 不存在，脚本以 exit 1 退出 → 测试失败 ✓（红）
  - **依赖:** Task 1.1

### 阶段 2: 实现（绿）

- [ ] **Task 2.1** — 修改 Ruby runner 块：`find_by` → `find_or_initialize_by`
  - **文件:** `gitService/scripts/sync_local_oauth_app_scopes.sh`（第 67-83 行）
  - **变更:** 将现有 Ruby runner 替换为：
    ```ruby
    uid = '${CLIENT_ID}'
    scopes = '${SCOPE}'
    redirect = '${REDIRECT_URI}'
    app = Doorkeeper::Application.find_or_initialize_by(uid: uid)
    if app.new_record?
      app.secret = '${CLIENT_SECRET}'
      app.name = 'gitOauth Local GitLab'
      app.redirect_uri = redirect
      app.scopes = scopes
      app.confidential = true
      puts "CREATED: #{app.name} scopes=#{app.scopes}"
    else
      if app.redirect_uri.to_s.strip.empty? && redirect.to_s.strip != ''
        app.redirect_uri = redirect
      end
      app.scopes = scopes
      puts "UPDATED: #{app.name} scopes=#{app.scopes}"
    end
    app.save!
    ```
  - **需要读取 `client_secret`:** 在原有基础上，Python YAML 解析部分需增加 `client_secret` 的读取（第 17-38 行新增 `CLIENT_SECRET` 变量）
  - **命令:** `bash gitService/scripts/sync_local_oauth_app_scopes.sh`
  - **依赖:** Task 1.1

- [ ] **Task 2.2** — 运行测试（预期通过—绿）
  - **命令:** `bash gitService/scripts/test_sync_oauth_app.sh`
  - **预期结果:**
    - 首次运行：创建 Application → 日志显示 `CREATED` → exit 0
    - 第二次运行：更新已存在的 Application → 日志显示 `UPDATED` → exit 0
    - Application 存在，scopes 与 YAML 一致
  - **依赖:** Task 2.1

### 阶段 3: 重构与打磨

- [ ] **Task 3.1** — 验证幂等性（NFR 可维护性 L2）
  - **命令:** `for i in 1 2 3; do bash gitService/scripts/sync_local_oauth_app_scopes.sh; done`
  - **检查:** Application count 不变，所有三次调用均 exit 0
  - **依赖:** Task 2.1

- [ ] **Task 3.2** — 验证日志输出区分 Created/Updated
  - **命令:** `bash gitService/scripts/sync_local_oauth_app_scopes.sh 2>&1 | grep -E "CREATED|UPDATED"`
  - **预期:** 首次调用显示 `CREATED`，后续显示 `UPDATED`
  - **依赖:** Task 2.1

- [ ] **Task 3.3** — ShellCheck 静态分析
  - **命令:** `shellcheck gitService/scripts/sync_local_oauth_app_scopes.sh`
  - **预期:** 无新增 warning（现有 warning 可接受）
  - **依赖:** Task 2.1

## 文件变更汇总

| 文件 | 操作 | 行数 |
|------|------|------|
| `gitService/scripts/sync_local_oauth_app_scopes.sh` | 修改 | ~20 行变更（Python YAML 解析 + Ruby runner） |
| `gitService/scripts/test_sync_oauth_app.sh` | 新建 | ~40 行（集成测试） |

## 质量关卡（来自 NFR）

| 关卡 | 验证方式 | 任务 |
|------|---------|------|
| 容错 L2 — 自愈创建 | Application 缺失时创建（非失败退出） | Task 2.2 |
| 可维护性 L2 — 幂等 | 重复执行不产生重复记录 | Task 3.1 |
| 可用性 L2 — 300s 超时 | 保持现有等待逻辑（第 50-65 行） | 不变 |

## 依赖图

```
Task 1.1 (测试脚本)
    ↓
Task 1.2 (确认红色)
    ↓
Task 2.1 (实现修复 + 读取 CLIENT_SECRET)
    ↓
Task 2.2 (确认绿色)
    ↓
Task 3.1 + Task 3.2 + Task 3.3 (并行打磨)
```

## 运行测试（CI 就绪）

```bash
# 单次运行
bash gitService/scripts/test_sync_oauth_app.sh

# 完整 NFR 验证
bash gitService/scripts/test_sync_oauth_app.sh && \
  for i in 1 2 3; do bash gitService/scripts/sync_local_oauth_app_scopes.sh || exit 1; done && \
  echo "所有检查通过"
```
