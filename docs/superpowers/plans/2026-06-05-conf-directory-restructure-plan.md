# conf/ 目录二级职能分组 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 conf/ 从扁平 21 目录重组为 `conf/<职能>/<服务>/` 二级结构（9 职能），通过 `git mv` 保留历史。

**Architecture:** 纯文件系统重组 — `git mv` 移动 21 个服务目录到 9 个职能子目录，更新 7 个 sync.manifest.yaml 的 `from` 相对路径，更新 runAll.yaml 的 12 处 `conf_app` 路径，更新 2 个 CI/脚本 glob 模式。配置值、端口、sync 协议均不变。

**Tech Stack:** Git (git mv), YAML (sync manifests, runAll), Bash (conf-sync-all.sh)

**Source of truth:** `docs/superpowers/specs/2026-06-05-conf-directory-restructure-design.md`

---

### Task 1: Git 迁移 — 创建职能目录并移动服务

**Files:**
- Move: `conf/task-auth/` → `conf/auth/task-auth/`
- Move: `conf/git-oauth/` → `conf/auth/git-oauth/`
- Move: `conf/core/django/` → `conf/core/django/`
- Move: `conf/task-gateway/` → `conf/gateway/task-gateway/`
- Move: `conf/task-container-gateway/` → `conf/gateway/task-container-gateway/`
- Move: `conf/task-sse/` → `conf/gateway/task-sse/`
- Move: `conf/ai-provider/` → `conf/ai/ai-provider/`
- Move: `conf/task-ai-endpoint/` → `conf/ai/task-ai-endpoint/`
- Move: `conf/task-agent-support/` → `conf/ai/task-agent-support/`
- Move: `conf/domain-events/` → `conf/events/domain-events/`
- Move: `conf/docker-infra/` → `conf/infra/docker-infra/`
- Move: `conf/git-service/` → `conf/infra/git-service/`
- Move: `conf/relay-to-trae/` → `conf/infra/relay-to-trae/`
- Move: `conf/task-bill/` → `conf/billing/task-bill/`
- Move: `conf/stripe/` → `conf/billing/stripe/`
- Move: `conf/vue/` → `conf/frontend/vue/`
- Move: `conf/mock-run-container/` → `conf/mock/mock-run-container/`
- Move: `conf/mock-trae-worker/` → `conf/mock/mock-trae-worker/`

- [ ] **Step 1: 创建 9 个职能目录**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
mkdir -p auth core gateway ai events infra billing frontend mock
```

- [ ] **Step 2: git mv 所有 21 个服务到对应职能**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf

# auth (认证授权)
git mv task-auth    auth/task-auth
git mv git-oauth    auth/git-oauth

# core (核心后端)
git mv django       core/django

# gateway (网关与实时通信)
git mv task-gateway           gateway/task-gateway
git mv task-container-gateway gateway/task-container-gateway
git mv task-sse               gateway/task-sse

# ai (AI 服务)
git mv ai-provider         ai/ai-provider
git mv task-ai-endpoint    ai/task-ai-endpoint
git mv task-agent-support  ai/task-agent-support

# events (领域事件)
git mv domain-events       events/domain-events

# infra (基础设施)
git mv docker-infra        infra/docker-infra
git mv git-service         infra/git-service
git mv relay-to-trae       infra/relay-to-trae

# billing (计费)
git mv task-bill           billing/task-bill
git mv stripe              billing/stripe

# frontend (前端)
git mv vue                 frontend/vue

# mock (Mock 服务)
git mv mock-run-container  mock/mock-run-container
git mv mock-trae-worker    mock/mock-trae-worker
```

- [ ] **Step 3: 验证 Git 正确识别为重命名（非 delete+add）**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
git status | head -50
```

Expected: 所有文件显示为 `renamed: conf/<old>/<file> -> conf/<职能>/<service>/<file>`（R flag），无 `deleted` + `new file` 配对。

- [ ] **Step 4: 验证新目录结构**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
ls -d */  | sort
```

Expected:
```
ai/  auth/  billing/  core/  events/  frontend/  gateway/  infra/  logs/  mock/
```

```bash
find . -maxdepth 3 -name "config.yaml" | sort
```

Expected: 21 个 config.yaml，全部在 `./<职能>/<服务>/config.yaml` 路径下。

- [ ] **Step 5: 确认根目录文件未被移动**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
ls -la README.md runAll.yaml .gitignore 2>&1
```

Expected: 三个文件均存在。

---

### Task 2: 更新 sync.manifest.yaml 路径

**Files:**
- Modify: `conf/ai/ai-provider/sync.manifest.yaml`
- Modify: `conf/core/django/sync.manifest.yaml`
- Modify: `conf/events/domain-events/sync.manifest.yaml`
- Modify: `conf/auth/git-oauth/sync.manifest.yaml`
- Modify: `conf/auth/task-auth/sync.manifest.yaml`
- Modify: `conf/gateway/task-sse/sync.manifest.yaml`
- Modify: `conf/frontend/vue/sync.manifest.yaml`

- [ ] **Step 1: 更新 ai-provider/sync.manifest.yaml**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
```

Edit `ai/ai-provider/sync.manifest.yaml`:
```yaml
version: "1"
fragments:
  - from: ../../core/django/config.yaml
    to: django.yaml
    pick: [ssoJwtSecret]
```

Note: `../django/` → `../../core/django/`

- [ ] **Step 2: 更新 core/django/sync.manifest.yaml**

Edit `core/django/sync.manifest.yaml`:
```yaml
version: "1"
fragments:
  - from: ../../auth/task-auth/config.yaml
    to: task-auth.yaml
    pick: [host, port, internalSecret, djangoInternalApiBase]
  - from: ../../events/domain-events/config.yaml
    to: domain-events.yaml
    pick: [transport, internalSecret]
  - from: ../../infra/docker-infra/config.yaml
    to: docker-infra.yaml
    pick: [redis, kafka]
```

Note:
- `../task-auth/` → `../../auth/task-auth/`
- `../domain-events/` → `../../events/domain-events/`
- `../docker-infra/` → `../../infra/docker-infra/`

- [ ] **Step 3: 更新 events/domain-events/sync.manifest.yaml**

Edit `events/domain-events/sync.manifest.yaml`:
```yaml
version: "1"
fragments:
  - from: ../../infra/docker-infra/config.yaml
    to: docker-infra.yaml
    pick: [redis, kafka]
  - from: ../../core/django/config.yaml
    to: django.yaml
    pick: [internalApiBase]
```

- [ ] **Step 4: 更新 auth/git-oauth/sync.manifest.yaml**

Edit `auth/git-oauth/sync.manifest.yaml`:
```yaml
version: "1"
fragments:
  - from: ../../core/django/config.yaml
    to: django.yaml
    pick: [ssoJwtSecret]
```

- [ ] **Step 5: 更新 auth/task-auth/sync.manifest.yaml**

Edit `auth/task-auth/sync.manifest.yaml`:
```yaml
version: "1"
fragments:
  - from: ../../core/django/config.yaml
    to: django.yaml
    pick: [host, port, internalApiBase]
```

- [ ] **Step 6: 更新 gateway/task-sse/sync.manifest.yaml**

Edit `gateway/task-sse/sync.manifest.yaml`:
```yaml
version: "1"
fragments:
  - from: ../../infra/docker-infra/config.yaml
    to: docker-infra.yaml
    pick: [redis, kafka]
```

- [ ] **Step 7: 更新 frontend/vue/sync.manifest.yaml**

Edit `frontend/vue/sync.manifest.yaml`:
```yaml
version: "1"
fragments:
  - from: ../../core/django/config.yaml
    to: django.yaml
    pick: [host, port, allowedHost, internalApiBase]
```

- [ ] **Step 8: 运行 conf-sync-all.sh 验证碎片生成**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
bash scripts/conf-sync-all.sh
```

Expected: exit 0，无错误输出。

- [ ] **Step 9: 验证碎片内容与迁移前一致**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
# 检查 GENERATED 头是否指向新路径
head -3 core/django/task-auth.yaml
head -3 core/django/domain-events.yaml
head -3 core/django/docker-infra.yaml
```

Expected: `# source:` 字段显示新的路径（如 `conf/auth/task-auth/config.yaml`）。

- [ ] **Step 10: Commit — Task 1+2**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
git add -A
git commit -m "refactor: restructure conf/ to conf/<function>/<service>/ hierarchy

- 9 functions: auth, core, gateway, ai, events, infra, billing, frontend, mock
- 21 services moved via git mv (history preserved)
- 7 sync.manifest.yaml from paths updated for new relative depth
- conf-sync-all.sh regenerated fragments verified identical"
```

---

### Task 3: 更新 runAll.yaml conf_app 路径

**Files:**
- Modify: `conf/runAll.yaml` (12 lines)

- [ ] **Step 1: 批量替换 conf_app 值**

Edit `conf/runAll.yaml` — update these lines:

| Line | Old | New |
|------|-----|-----|
| 103 | `conf_app: task-auth` | `conf_app: auth/task-auth` |
| 118 | `conf_app: task-bill` | `conf_app: billing/task-bill` |
| 141 | `conf_app: ai-provider` | `conf_app: ai/ai-provider` |
| 158 | `conf_app: django` | `conf_app: core/django` |
| 179 | `conf_app: task-gateway` | `conf_app: gateway/task-gateway` |
| 191 | `conf_app: task-agent-support` | `conf_app: ai/task-agent-support` |
| 204 | `conf_app: task-ai-endpoint` | `conf_app: ai/task-ai-endpoint` |
| 217 | `conf_app: task-container-gateway` | `conf_app: gateway/task-container-gateway` |
| 230 | `conf_app: task-sse` | `conf_app: gateway/task-sse` |
| 242 | `conf_app: vue` | `conf_app: frontend/vue` |
| 547 | `conf_app: mock-run-container` | `conf_app: mock/mock-run-container` |
| 560 | `conf_app: relay-to-trae` | `conf_app: infra/relay-to-trae` |

- [ ] **Step 2: 验证 runAll.yaml 语法有效**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
python3 -c "import yaml; yaml.safe_load(open('conf/runAll.yaml')); print('YAML valid')"
```

Expected: `YAML valid`

- [ ] **Step 3: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
git add runAll.yaml
git commit -m "refactor: update conf_app paths for conf/<function>/<service>/ hierarchy"
```

---

### Task 4: 更新 CI 脚本和工具路径

**Files:**
- Modify: `scripts/conf-sync-all.sh`
- Modify: `scripts/ci/check_conf_sync.sh`
- Modify: `task2app/paths.conf` (if FindMonorepoRoot references conf/core/django/)

- [ ] **Step 1: 检查 conf-sync-all.sh 的 glob 模式**

```bash
grep -n "conf/" /Users/task2app/gitClone/ramDisk/ram-mount/scripts/conf-sync-all.sh
```

- [ ] **Step 2: 更新 glob 从 `conf/*/sync.sh` → `conf/*/*/sync.sh`**

If the script uses `conf/*/sync.sh` (one-level glob), change to `conf/*/*/sync.sh` (two-level glob) to match the new `conf/<职能>/<服务>/sync.sh` structure.

Note: `conf/logs/` 不含 sync.sh，glob 不会误匹配。如果担心，可用 `conf/*/*/sync.sh` 精确匹配两层目录。

- [ ] **Step 3: 同样更新 check_conf_sync.sh**

```bash
grep -n "conf/" /Users/task2app/gitClone/ramDisk/ram-mount/scripts/ci/check_conf_sync.sh
```

Update any `conf/*/` globs to `conf/*/*/`.

- [ ] **Step 4: 检查 FindMonorepoRoot 标记**

```bash
grep -rn "conf/core/django/config.yaml\|conf/core/django" /Users/task2app/gitClone/ramDisk/ram-mount/task2app/paths.conf 2>/dev/null || echo "paths.conf not found or no reference"
grep -rn "conf/core/django" /Users/task2app/gitClone/ramDisk/ram-mount/task2app/Saas_project/saas_project/settings_test.py 2>/dev/null | head -5
```

If `conf/core/django/` is referenced as a monorepo root marker, update to `conf/core/django/`.

- [ ] **Step 5: 检查 conf-read.py 是否有硬编码路径**

```bash
grep -n "conf/" /Users/task2app/gitClone/ramDisk/ram-mount/scripts/conf-read.py 2>/dev/null | head -20
```

Update any hardcoded `conf/<service>/` paths. If conf-read.py uses `conf_app` from runAll.yaml dynamically, it auto-adapts (no change needed).

- [ ] **Step 6: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
git add scripts/conf-sync-all.sh scripts/ci/check_conf_sync.sh
# add other modified files if any
git commit -m "refactor: update CI globs and paths for conf/<function>/<service>/"
```

---

### Task 5: 运行时验证

- [ ] **Step 1: 运行 conf-sync-all.sh**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
bash scripts/conf-sync-all.sh
```

Expected: exit 0, all syncs pass.

- [ ] **Step 2: 运行 CI conf-sync 检查**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
bash scripts/ci/check_conf_sync.sh
```

Expected: exit 0.

- [ ] **Step 3: 验证 conf-read.py 通过新路径读取**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
python3 scripts/conf-read.py core/django port
python3 scripts/conf-read.py auth/task-auth port
```

Expected: 返回正确的端口号（与迁移前相同）。

- [ ] **Step 4: Commit (if any verification fixup needed)**

---

### Task 6: 文档更新

**Files:**
- Modify: `conf/README.md`
- Modify: `value-stream.yaml` (description strings only)

- [ ] **Step 1: 更新 conf/README.md**

Update `conf/README.md`:
- 第 13-17 行的 Layout 部分：更新为新的二级目录结构
- 路径引用：`conf/core/django/` → `conf/core/django/` 等

- [ ] **Step 2: 更新 value-stream.yaml 描述**

```bash
grep -n "conf/" /Users/task2app/gitClone/ramDisk/ram-mount/value-stream.yaml | grep -v "^[0-9]*:.*#" | head -20
```

Update description strings that reference `conf/<service>/` paths to new paths.

- [ ] **Step 3: Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
git add conf/README.md value-stream.yaml
git commit -m "docs: update conf path references for function/service hierarchy"
```

---

### Task 7: 最终验证

- [ ] **Step 1: 全量路径引用检查**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount
# 不应存在旧路径的代码引用（排除 .git/、docs/superpowers/specs/ 历史设计文档）
grep -r "conf/core/django/" --include="*.py" --include="*.sh" --include="*.yaml" --include="*.yml" --include="*.md" \
  --exclude-dir=.git --exclude-dir=docs/superpowers/specs . | grep -v "Binary" | head -20
```

Expected: 无输出（或仅为 docs/superpowers/specs/ 中的历史设计文档引用）。

- [ ] **Step 2: 验证 Git log 连续性**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
git log --oneline -5
```

Expected: 迁移 commits 在历史中可见。

- [ ] **Step 3: 最终 Commit**

```bash
cd /Users/task2app/gitClone/ramDisk/ram-mount/conf
git status
```

Expected: clean working tree.

---

## 概览

| Task | 描述 | 文件数 | 风险 |
|------|------|--------|------|
| 1 | Git 迁移 (git mv) | 21 moves | 低 |
| 2 | Sync manifest 路径更新 | 7 edits | 中 |
| 3 | runAll.yaml conf_app | 12 lines | 中 |
| 4 | CI/脚本路径 | ~3 files | 中 |
| 5 | 运行时验证 | — | 低 |
| 6 | 文档更新 | 2 files | 低 |
| 7 | 最终验证 | — | 低 |
