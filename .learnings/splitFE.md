# splitFE — task2app/front_project → taskFE 拆分执行方案

> **目标**：将 `task2app/front_project` 从 `task2app` 目录中独立为顶层 `taskFE/`，让 `task2app` 仅提供 API（Django），`taskFE` 提供前端（Vue 3 + Vite）。
>
> **设计日期**：2026-07-27
> **作者**：claude / ljy
> **状态**：待执行

---

## 前置条件

- [ ] 所有工作区 clean（`git status` 无未提交更改）
- [ ] `task2app/front_project/app/node_modules/` 已 `npm install`
- [ ] 无正在运行的 Vite dev server / build watch 进程
- [ ] Django `runserver` 已停止

---

## Phase 0 — 准备工作

### 0.1 确认当前目录结构

```bash
git status
ls -la task2app/front_project/
ls -la task2app/Saas_project/frontend_app/
```

### 0.2 停止所有相关进程

```bash
pkill -f 'vite' 2>/dev/null || true
pkill -f 'node.*vite' 2>/dev/null || true
lsof -ti:4000 2>/dev/null | xargs kill -9 2>/dev/null || true
```

---

## Phase 1 — 物理文件移动（最关键步骤）

### 1.1 创建 taskFE 目录并移动文件

```bash
# 在仓库根目录执行
cd /tmp/ram-work

# 创建 taskFE 目录结构
mkdir -p taskFE

# 用 git mv 移动所有文件（保留 git 历史）
git mv task2app/front_project/app        taskFE/app
git mv task2app/front_project/docs       taskFE/docs
git mv task2app/front_project/static     taskFE/static
git mv task2app/front_project/node_modules taskFE/node_modules 2>/dev/null || true
git mv task2app/front_project/package.json       taskFE/package.json 2>/dev/null || true
git mv task2app/front_project/package-lock.json  taskFE/package-lock.json 2>/dev/null || true

# 如果 front_project 下还有其他零散文件，一并移动
# 最后清理空目录
rmdir task2app/front_project 2>/dev/null || true
```

### 1.2 更新 taskFE 内部的相对路径引用

> ⚠️ **最关键**：`vite.config.js` 中的 `monorepoRoot` 计算从 3 级变为 2 级

**文件**: `taskFE/app/vite.config.js` (line ~56)

```javascript
// 旧：从 task2app/front_project/app/ 向上 3 级到 repo 根
const monorepoRoot = resolve(__dirname, '../../../')
// 新：从 taskFE/app/ 向上 2 级到 repo 根
const monorepoRoot = resolve(__dirname, '../../')
```

**文件**: `taskFE/app/scripts/collectstatic-after-vite.sh`

```bash
# 旧
task2app_root="$(cd "$script_dir/../../.." && pwd)"
# 新：scripts/ → app/ → taskFE/ → repo root
task2app_root="$(cd "$script_dir/../.." && pwd)"   # 变量名保留不变，避免大面积改动
# ↑ task2app_root 实际上是 repo root，名称保留以免连锁改动，语义上应理解为 "repo_root"
```

同时更新脚本中的路径引用：
```bash
# 旧
vite_assets_src="$task2app_root/front_project/static/assets"
# 新
vite_assets_src="$task2app_root/taskFE/static/assets"

# 旧
vite_assets_src="$task2app_root/front_project/app/static/assets"
# 新
vite_assets_src="$task2app_root/taskFE/app/static/assets"
```

**文件**: `taskFE/app/scripts/runall-lifecycle.sh` — 无需改，`cd "$app_dir"` 后用相对路径。

**文件**: `taskFE/app/package.json` — 保留不变。前端自身的 npm scripts 使用相对路径 `../..` 指向 playwright 和 task2app：

```json
"test:e2e": "npm --prefix ../../playwright run test:e2e",
```
由于 taskFE 现在在 repo 根下，路径应该是 `../task2app/playwright`，修改为：
```json
"test:e2e": "npm --prefix ../task2app/playwright run test:e2e",
"test:e2e:headed": "npm --prefix ../task2app/playwright run test:e2e:headed",
"test:e2e:quick:taskdetail": "npm --prefix ../task2app/playwright run test:e2e:quick:taskdetail",
"verify:daydaymoney-api-host": "npm --prefix ../task2app/playwright run verify:daydaymoney-api-host"
```

---

## Phase 2 — task2app 内部路径更新

### 2.1 `task2app/run.sh` (影响最大：29 处引用)

所有 `$SCRIPT_DIR/front_project/` → `$SCRIPT_DIR/../taskFE/`

```bash
# Line 86-87 (PID 和 LOG 变量 — 变量名保留以免连锁改名)
FRONT_PROJECT_BUILD_WATCH_PID_FILE="$LOG_DIR/front_project_build_watch.pid"   # 不变
FRONT_PROJECT_BUILD_WATCH_LOG="$LOG_DIR/front_project_build_watch.log"        # 不变

# Line 359-368 (cleanup 路径)
rm -rf "$SCRIPT_DIR/../taskFE/app/.vite"
rm -rf "$SCRIPT_DIR/../taskFE/app/static/vue"
rm -rf "$SCRIPT_DIR/../taskFE/dist/"
rm -rf "$SCRIPT_DIR/../taskFE/app/node_modules/.cache"

# Line 382 (构建)
cd "$SCRIPT_DIR/../taskFE/app" && npm run build >> "$MAIN_LOG" 2>&1

# Line 447 (VUE_PROJECT_DIR)
VUE_PROJECT_DIR="$SCRIPT_DIR/../taskFE/app"

# Line 552 (vite_manifest)
local vite_manifest="$SCRIPT_DIR/../taskFE/static/assets/.vite/manifest.json"

# Line 572 (pkill inotifywait pattern)
pkill -f 'inotifywait.*taskFE/static/assets' 2>/dev/null || true

# Line 585 (build watch)
cd "$SCRIPT_DIR/../taskFE/app" && npm run build:vite -- --watch &

# Line 587 (assets_dir)
assets_dir="$SCRIPT_DIR/../taskFE/static/assets"
```

以及所有包含 `front_project` 的 echo/注释字符串更新。

### 2.2 `task2app/Saas_project/saas_project/settings.py`

```python
# ~Line 629-631
# 旧
_vue_static = Path(BASE_DIR).parent / 'front_project' / 'static'
_vue_assets = _vue_static / 'assets'
_vue_assets_app = Path(BASE_DIR).parent / 'front_project' / 'app' / 'static' / 'assets'
# 新：BASE_DIR = Saas_project/, parent = task2app/, parent.parent = repo_root
_vue_static = Path(BASE_DIR).parent.parent / 'taskFE' / 'static'
_vue_assets = _vue_static / 'assets'
_vue_assets_app = Path(BASE_DIR).parent.parent / 'taskFE' / 'app' / 'static' / 'assets'
```

### 2.3 `task2app/Saas_project/frontend_app/templatetags/vite_tags.py`

```python
# ~Line 39-41 (_front_project_app_src_dir)
# 旧
return Path(settings.BASE_DIR).parent / 'front_project' / 'app' / 'src'
# 新
return Path(settings.BASE_DIR).parent.parent / 'taskFE' / 'app' / 'src'
```

```python
# ~Line 88-94 (vite_asset manifest paths)
# 旧
Path(settings.BASE_DIR).parent / 'front_project' / 'static' / 'assets' / 'manifest.json',
Path(settings.BASE_DIR).parent / 'front_project' / 'static' / 'assets' / '.vite' / 'manifest.json',
Path(settings.BASE_DIR).parent / 'front_project' / 'app' / 'static' / 'assets' / 'manifest.json',
Path(settings.BASE_DIR).parent / 'front_project' / 'app' / 'static' / 'assets' / '.vite' / 'manifest.json',
# 新
Path(settings.BASE_DIR).parent.parent / 'taskFE' / 'static' / 'assets' / 'manifest.json',
Path(settings.BASE_DIR).parent.parent / 'taskFE' / 'static' / 'assets' / '.vite' / 'manifest.json',
Path(settings.BASE_DIR).parent.parent / 'taskFE' / 'app' / 'static' / 'assets' / 'manifest.json',
Path(settings.BASE_DIR).parent.parent / 'taskFE' / 'app' / 'static' / 'assets' / '.vite' / 'manifest.json',
```

### 2.4 `task2app/Saas_project/frontend_app/templates/frontend/spa.html`

无需修改。模板由 Django 内部渲染，`vite_tags.py` 的路径变更是透明的。

### 2.5 `task2app/playwright/package.json`

> 已确认：`task2app/playwright/front_project/` 是 `task2app/playwright/` 下的独立目录（非 front_project 子目录）。迁移后 playwright config/scripts/tests 全部移至 `taskFE/`（见 Phase 5）。

```json
// 旧（working dir 为 task2app/playwright/，通过相对路径引用同目录下的 front_project/）
"test:e2e": "playwright test -c front_project/playwright.config.js --timeout=60000",
"test:e2e:headed": "playwright test -c front_project/playwright.config.js --timeout=60000 --headed",
"test:e2e:quick:taskdetail": "playwright test -c front_project/playwright.config.js --timeout=60000 front_project/tests/TaskDetail.layer-agent-steps-ui-mock.playwright.test.js",
"verify:daydaymoney-api-host": "node front_project/scripts/verify-daydaymoney-login-api-host.mjs",
"verify:git-oauth-cdp": "node front_project/tests/verify-git-oauth-playwright-cdp.mjs",
"daydaymoney:open-task-vscode": "node front_project/scripts/daydaymoney-open-task-vscode.mjs",
"daydaymoney:verify-ztree-mock-start": "node front_project/scripts/daydaymoney-verify-ztree-mock-start.mjs",
"test:e2e:remote-login": "playwright test -c front_project/playwright.verify.config.js front_project/tests/AuthLogin.remote-email-e2e.playwright.test.js --project=chromium"

// 新（config/scripts/tests 都在 taskFE/，从 task2app/playwright/ 出发用 ../../taskFE/）
"test:e2e": "playwright test -c ../../taskFE/playwright.config.js --timeout=60000",
"test:e2e:headed": "playwright test -c ../../taskFE/playwright.config.js --timeout=60000 --headed",
"test:e2e:quick:taskdetail": "playwright test -c ../../taskFE/playwright.config.js --timeout=60000 ../../taskFE/tests/TaskDetail.layer-agent-steps-ui-mock.playwright.test.js",
"verify:daydaymoney-api-host": "node ../../taskFE/scripts/verify-daydaymoney-login-api-host.mjs",
"verify:git-oauth-cdp": "node ../../taskFE/tests/verify-git-oauth-playwright-cdp.mjs",
"daydaymoney:open-task-vscode": "node ../../taskFE/scripts/daydaymoney-open-task-vscode.mjs",
"daydaymoney:verify-ztree-mock-start": "node ../../taskFE/scripts/daydaymoney-verify-ztree-mock-start.mjs",
"test:e2e:remote-login": "playwright test -c ../../taskFE/playwright.verify.config.js ../../taskFE/tests/AuthLogin.remote-email-e2e.playwright.test.js --project=chromium"
```

### 2.6 `task2app/CLAUDE.md`

```markdown
- **前端**: `front_project/` (React/Vue/Node.js)  →  `../taskFE/` (Vue 3 + Vite)
```

以及测试路径：
```markdown
- 后端测试: `tests/`，前端 E2E: `playwright/front_project/tests/`  →  `../taskFE/tests/`
```

### 2.7 `task2app/Saas_project/scripts/auxiliary/verify_security_group_ajax.py`

搜索此文件中的 `front_project` 引用并更新路径（如有）。

### 2.8 `task2app/Saas_project/tests/test_vite_tags_cache_bust.py`

搜索此文件中的 `front_project` 引用并更新路径（如有）。

---

## Phase 3 — monorepo 配置文件更新

### 3.1 `conf/runAll.yaml`

```yaml
# ~Line 376
# 旧
working_dir: task2app/front_project/app
# 新
working_dir: taskFE/app
```

`conf_app` 字段 `frontend/vue` 保持不变（配置文件在 `conf/frontend/vue/` 下，不移动）。

### 3.2 `conf/value-stream.yaml` (约 31 处引用)

> ⚠️ 这是引用数量最多的配置文件。所有路径都是相对于 `runner.working_dir` 的。

需要批量替换的路径模式：

| 旧模式 | 新模式 | 说明 |
|--------|--------|------|
| `../front_project/app/src/...` | `../../taskFE/app/src/...` | working_dir 为 `task2app/Saas_project`，原向上一级到 `task2app/` → 现需向上两级到 repo root |
| `../../task2app/front_project/app/...` | `../../taskFE/app/...` | working_dir 为 repo root 或顶级目录 |
| `../../task2app/playwright/front_project/...` | `../../taskFE/...` | playwright 已移至 taskFE（见 Phase 5） |
| `../playwright/front_project/...` | `../../taskFE/...` | playwright 已移至 taskFE |
| `../front_project/docs/...` | `../../taskFE/docs/...` | 文档路径 |
| `front_project/tests/...` | `../../taskFE/tests/...` | 直接路径 |

**推荐使用 sed 批量替换**：
```bash
# 在 conf/value-stream.yaml 中
# 1. 先处理 ../../task2app/front_project → ../../taskFE
sed -i 's|../../task2app/front_project/|../../taskFE/|g' conf/value-stream.yaml

# 2. 处理 ../front_project → ../../taskFE (从 task2app/Saas_project 出发)
sed -i 's|../front_project/|../../taskFE/|g' conf/value-stream.yaml

# 3. 处理 playwright 路径（已迁移到 taskFE）
sed -i 's|../../task2app/playwright/front_project/|../../taskFE/|g' conf/value-stream.yaml
sed -i 's|../playwright/front_project/|../../taskFE/|g' conf/value-stream.yaml

# 4. 处理 front_project/tests → ../../taskFE/tests (直接路径)
sed -i 's|front_project/tests/|../../taskFE/tests/|g' conf/value-stream.yaml
```

### 3.3 `conf/value-stream.yaml.ai.md`

```bash
# 文档性引用，同样替换
sed -i 's|task2app/front_project/|taskFE/|g' conf/value-stream.yaml.ai.md
```

### 3.4 `conf/runAll.yaml.ai.md`

```bash
sed -i 's|task2app/front_project/app|taskFE/app|g' conf/runAll.yaml.ai.md
```

### 3.5 `conf/README.md`

```markdown
- `frontend/vue/` — front_project (Vite)  →  `frontend/vue/` — taskFE (Vite)
```

### 3.6 `conf/port_config.json.md`

搜索 `front_project` 并更新：
```markdown
与 `front_project` 的 Vite 代理目标  →  与 `taskFE` 的 Vite 代理目标
```

---

## Phase 4 — 其他仓库级文件更新

### 4.1 `runAll/src/service_language_test.go`

```go
// Line 64
// 旧
WorkingDir: filepath.Join(repoRoot, "task2app", "front_project", "app"),
// 新
WorkingDir: filepath.Join(repoRoot, "taskFE", "app"),
```

### 4.2 `shareLib/autorunstartvm/startvm_request.go`

仅注释引用，更新注释中的 `front_project` → `taskFE`（2 处）。

### 4.3 `.ai/` 失败经验文档 (~15 处)

```bash
# 批量替换所有 .ai/ 下的 front_project 引用
find .ai -name "*.md" -exec sed -i 's|task2app/front_project/|taskFE/|g' {} +
```

### 4.4 `.learnings/` 历史文档 (~20 处)

```bash
# 批量替换所有 .learnings/ 下的 front_project 引用
find .learnings -name "*.md" -exec sed -i 's|task2app/front_project/|taskFE/|g' {} +
```

> ⚠️ 注意：`.learnings/splitFE.md` 本身不替换（本文件是新创建的）。

---

## Phase 5 — Playwright 配置迁移至 taskFE

### 5.1 现状确认

`task2app/playwright/front_project/` 是 `task2app/playwright/` 下的**独立子目录**（非 symlink），结构：

```
task2app/playwright/
├── helpers/
│   └── loadConfYaml.mjs              ← 共享工具（playwright config 引用此文件）
├── package.json                      ← npm scripts 引用 front_project/
├── debug-login.mjs
├── diagnose_cors.mjs
└── front_project/                    ← 🎯 迁移目标
    ├── playwright.config.js          ← 引用 ../helpers/loadConfYaml.mjs
    ├── playwright.verify.config.js   ← 引用 ../helpers/loadConfYaml.mjs
    ├── playwright.*.config.js        ← 11 个配置文件变体
    ├── scripts/                      ← 8 个 .mjs 脚本
    └── tests/                        ← 大量 .playwright.test.js 测试
```

关键耦合点（`playwright.config.js`）：
```javascript
const __dirname = path.dirname(__filename);                          // ...playwright/front_project/
const vueAppRoot = path.resolve(__dirname, '../../front_project/app');  // → task2app/front_project/app [需修正]
const djangoProjectRoot = path.resolve(__dirname, '../../Saas_project'); // → task2app/Saas_project [需修正]
// ...
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';  // → playwright/helpers/ [需修正]
```

### 5.2 迁移操作

```bash
# 将 playwright 配置、脚本、测试全部移到 taskFE 下
git mv task2app/playwright/front_project/playwright.config.js             taskFE/playwright.config.js
git mv task2app/playwright/front_project/playwright.verify.config.js      taskFE/playwright.verify.config.js
git mv task2app/playwright/front_project/playwright.chromium.config.js    taskFE/playwright.chromium.config.js
git mv task2app/playwright/front_project/playwright.config.cdp.js         taskFE/playwright.config.cdp.js
git mv task2app/playwright/front_project/playwright.config.chromium.js    taskFE/playwright.config.chromium.js
git mv task2app/playwright/front_project/playwright.config.headless.js    taskFE/playwright.config.headless.js
git mv task2app/playwright/front_project/playwright.config.local.js       taskFE/playwright.config.local.js
git mv task2app/playwright/front_project/playwright.config.oauth-test.js  taskFE/playwright.config.oauth-test.js
git mv task2app/playwright/front_project/playwright.config.ssh-branch-preview.js taskFE/playwright.config.ssh-branch-preview.js
git mv task2app/playwright/front_project/playwright.verify.chromium.config.js   taskFE/playwright.verify.chromium.config.js
git mv task2app/playwright/front_project/playwright.config.js.ai.md       taskFE/playwright.config.js.ai.md
git mv task2app/playwright/front_project/scripts/                         taskFE/scripts/
git mv task2app/playwright/front_project/tests/                           taskFE/tests/

# 清理空目录
rmdir task2app/playwright/front_project/ 2>/dev/null || true
```

### 5.3 路径修正（执行迁移后）

**文件**: `taskFE/playwright.config.js`

```javascript
// 旧
const vueAppRoot = path.resolve(__dirname, '../../front_project/app');
const djangoProjectRoot = path.resolve(__dirname, '../../Saas_project');
// 新：从 taskFE/ 出发
const vueAppRoot = path.resolve(__dirname, './app');
const djangoProjectRoot = path.resolve(__dirname, '../task2app/Saas_project');
```

```javascript
// 旧
import { loadPortConfig, clientReachableHost } from '../helpers/loadConfYaml.mjs';
// 新
import { loadPortConfig, clientReachableHost } from '../task2app/playwright/helpers/loadConfYaml.mjs';
```

**文件**: `taskFE/playwright.verify.config.js` — 同样更新 import 路径：
```javascript
// 旧
import { loadPortConfig, clientReachableHost } from '../helpers/loadConfYaml.mjs';
// 新
import { loadPortConfig, clientReachableHost } from '../task2app/playwright/helpers/loadConfYaml.mjs';
```

**文件**: `taskFE/playwright.config.js` — `testDir` 保持不变（`./tests`，因为 tests/ 现在与 config 在同一目录下）。

**文件**: 所有其他 playwright config 变体 — 同样检查并修正 `vueAppRoot`、`djangoProjectRoot`、`helpers` import 路径。

---

## Phase 6 — 验证步骤

### 6.1 构建验证

```bash
# 1. Vite 生产构建
cd taskFE/app && npm run build
# 预期：构建成功，taskFE/static/assets/ 下有 main-*.js 和 manifest.json

# 2. Django collectstatic
cd task2app && source activate_env.sh && cd Saas_project
python3 manage.py collectstatic --noinput
# 预期：收集成功，collected_static/assets/ 下有以下划线 hash 的 JS/CSS
```

### 6.2 开发模式验证

```bash
# 1. 启动 Vite dev server
cd taskFE/app && npm run dev &
# 预期：Vite 启动在配置的端口（默认 4000），/health 返回 200

# 2. 访问健康检查
curl http://localhost:4000/health
# 预期：{"service":"taskFE","ok":true,"checks":{}}

# 3. 启动 Django
cd task2app && bash run.sh
# 预期：Django 正常启动，SPA 页面可访问
```

### 6.3 静态资源验证

```bash
# 确认 SPA 页面加载 JS/CSS 200
curl -s http://localhost:8001/ | grep -o 'src="[^"]*"' | head -5
# 预期：JS/CSS 引用路径正确（/static/assets/main-*.js）

# 确认关键资源可访问
curl -sI http://localhost:8001/static/assets/main-*.js | head -3
# 预期：HTTP/1.1 200 OK, Content-Type: text/javascript
```

### 6.4 Playwright E2E 验证

```bash
# 从 task2app/playwright/ 执行
cd task2app/playwright && npm run test:e2e
# 预期：测试通过（或至少错误指向路径问题而非功能回退）
```

### 6.5 runAll 集成验证

```bash
# build 模式
cd /tmp/ram-work && bash runAll/run.sh build taskFE
# 预期：构建成功

# start 模式
bash runAll/run.sh start taskFE
# 预期：taskFE healthy
```

### 6.6 全局搜索残留引用

```bash
# 确保没有残留的 front_project 路径引用
grep -rn "front_project" . --include="*.py" --include="*.sh" --include="*.js" --include="*.yaml" --include="*.yml" --include="*.go" --include="*.json" --include="*.md" \
  | grep -v node_modules | grep -v ".git/" | grep -v ".pytest_cache" | grep -v "test-results" | grep -v "logs/" | grep -v "__pycache__" | grep -v "splitFE.md" \
  | grep -v "\.learnings/"  # .learnings/ 历史文档可以保留旧引用（历史记录）
# 预期：仅 .learnings/ 历史文档中有残留（可接受），其他关键路径全部更新
```

---

## Phase 7 — 提交

```bash
git add -A
git status  # 确认变更范围

git commit -m "refactor: 拆分 front_project → taskFE 独立前端目录

将 Vue 3/Vite 前端从 task2app/front_project 独立为顶层 taskFE/
- 物理移动：854 源文件 + 构建配置
- 更新 30+ 文件路径引用：Django STATICFILES_DIRS、run.sh、runAll.yaml、value-stream.yaml、vite.config.js
- vite.config.js monorepoRoot 计算修正（3→2 级）
- Playwright E2E 配置随前端迁移至 taskFE/
- conf/ 配置文件、Go 测试、.ai 文档同步更新

task2app 现在仅提供 API 功能（Django + frontend_app SPA shell）

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## 回滚方案

如果拆分后出现不可修复的问题：

```bash
git revert <commit_hash>
# 或
git reset --hard HEAD~1
```

---

## 风险矩阵

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|---------|
| `vite.config.js` `monorepoRoot` 计算错误 → Vite 无法读取配置 | 高 | 高 | 手动验证 `resolve(__dirname, '../../')` 指向 repo 根 |
| Django `STATICFILES_DIRS` 路径错误 → 生产 SPA 404 | 中 | 严重 | Phase 6.3 验证 |
| `value-stream.yaml` 路径遗漏 | 中 | 中 | Phase 6.6 全局搜索 |
| `playwright/front_project/` 非 front_project 子目录 → 路径分析错误 | 中 | 中 | Phase 5.1 确认 |
| 函数名 `kill_front_project_build_watch_if_any` 保留 → 混淆 | 低 | 低 | 后续迭代可改名，功能不受影响 |

---

## 执行清单（供智能体逐项勾选）

- [ ] Phase 0: 停止进程、确认 clean 状态
- [ ] Phase 1.1: `git mv` 移动 front_project → taskFE
- [ ] Phase 1.2: 更新 `vite.config.js` monorepoRoot (3→2)
- [ ] Phase 1.2: 更新 `collectstatic-after-vite.sh`
- [ ] Phase 1.2: 更新 `package.json` playwright 路径
- [ ] Phase 2.1: 更新 `task2app/run.sh` (29 处)
- [ ] Phase 2.2: 更新 `settings.py` STATICFILES_DIRS
- [ ] Phase 2.3: 更新 `vite_tags.py` 全部路径
- [ ] Phase 2.5: 更新 `task2app/playwright/package.json`
- [ ] Phase 2.6: 更新 `task2app/CLAUDE.md`
- [ ] Phase 3.1: 更新 `conf/runAll.yaml` working_dir
- [ ] Phase 3.2: 更新 `conf/value-stream.yaml` (~31 处)
- [ ] Phase 3.3-3.6: 更新 conf 其他文档
- [ ] Phase 4.1: 更新 `runAll/src/service_language_test.go`
- [ ] Phase 4.2: 更新 `shareLib/autorunstartvm/startvm_request.go`
- [ ] Phase 4.3-4.4: 批量更新 `.ai/` 和 `.learnings/` 路径
- [ ] Phase 5: Playwright 配置迁移
- [ ] Phase 6.1: 构建验证 (vite build + collectstatic)
- [ ] Phase 6.2: 开发模式验证 (dev server + Django)
- [ ] Phase 6.3: 静态资源验证 (200 OK)
- [ ] Phase 6.4: Playwright E2E 验证
- [ ] Phase 6.5: runAll 集成验证
- [ ] Phase 6.6: 全局搜索残留引用
- [ ] Phase 7: Git commit
