# 将 conf-read.py / conf-sync.py 合并进 runAll 服务

> 头脑风暴产物 — 设计文档
> 日期: 2026-06-05
> 状态: 待审批

---

## 一、动机

`conf-read.py` 和 `conf-sync.py` 是 monorepo 配置管理的基础设施。它们不属于任何业务服务，而是 **编排层的工具**。

runAll 已是 monorepo 的编排器（启动/停止所有服务、管理生命周期、同步配置），且 `runAll/src/conf_sync.go` 已直接调用 `scripts/conf-sync-all.sh`。将配置脚本合并进 runAll 是架构的收束——编排器拥有配置管理能力。

---

## 二、现状分析

### 涉及的文件

| 文件 | 当前位置 | 语言 | 行数 | 依赖 |
|------|----------|------|------|------|
| `conf-read.py` | scripts/ | Python | 98 | `task2app/Saas_project/config/conf_loader.py` |
| `conf-sync.py` | scripts/ | Python | 85 | `scripts/conf_lib.py` |
| `conf_lib.py` | scripts/ | Python | 71 | PyYAML |
| `conf-sync-all.sh` | scripts/ | Bash | 11 | `conf-sync.py` |
| `migrate_port_config_to_conf.py` | scripts/ | Python | 106 | `scripts/conf_lib.py` |

### 全部调用者（需路径更新）

```
conf-read.py
├── taskSSE/src/config.mjs:67    ← execSync('scripts/conf-read.py' ...)
├── taskSSE/src/config.mjs:78    ← execSync('scripts/conf-read.py' snapshot-json)
├── 开发者 CLI                   ← python3 scripts/conf-read.py ...
└── conf/README.md               ← 文档引用

conf-sync.py
├── conf/core/django/sync.sh     ← exec python3 "$ROOT/scripts/conf-sync.py" django
├── conf/frontend/vue/sync.sh    ← exec python3 "$ROOT/scripts/conf-sync.py" vue
├── conf/auth/task-auth/sync.sh  ← exec python3 "$ROOT/scripts/conf-sync.py" task-auth
├── conf/auth/git-oauth/sync.sh  ← exec python3 "$ROOT/scripts/conf-sync.py" git-oauth
├── conf/events/domain-events/sync.sh
├── conf/ai/ai-provider/sync.sh
├── conf/gateway/task-sse/sync.sh
└── conf/core/django/sync.sh

conf-sync-all.sh
├── runAll/src/conf_sync.go:39   ← bash scripts/conf-sync-all.sh
└── scripts/ci/check_conf_sync.sh:33 ← ./scripts/conf-sync-all.sh
```

### 当前架构的痛点

```
conf-read.py ───import──▶ task2app/Saas_project/config/conf_loader.py
     ▲                         ▲
     │                         │
  scripts/               task2app/
                         
  问题：配置读取脚本（scripts/）反向依赖业务项目（task2app/）
```

此外 `conf_lib.py` 和 `task2app/.../conf_loader.py` 存在 **函数重复**（`deep_merge`、`load_yaml`、`repo_root` 各有两套实现）。

---

## 三、设计方案

### 目标结构

```
runAll/
├── bin/runAll                    # Go 编译产物（不变）
├── src/
│   ├── conf_sync.go              # 更新脚本路径引用
│   └── ...                       # 其他 Go 源文件
├── scripts/
│   ├── hooks/                    # 已有
│   ├── conf_loader.py            # 新：自包含的 YAML 配置加载器
│   ├── conf_lib.py               # 迁移自 scripts/conf_lib.py
│   ├── conf-read.py              # 迁移自 scripts/conf-read.py
│   ├── conf-sync.py              # 迁移自 scripts/conf-sync.py
│   ├── conf-sync-all.sh          # 迁移自 scripts/conf-sync-all.sh
│   └── migrate_port_config_to_conf.py  # 迁移自 scripts/
├── run.sh
├── build.sh
└── ...
```

### 核心决策：解除 task2app 依赖

当前 `conf-read.py` 通过 `sys.path.insert` 从 `task2app/Saas_project/config/conf_loader.py` 导入以下函数：

| 函数 | 用途 | 处理方式 |
|------|------|----------|
| `JSON_KEY_TO_APP_DIR` | app 名 → conf/ 路径映射 | 移到 `runAll/scripts/conf_loader.py` |
| `monorepo_root()` | 定位仓库根目录 | 移到 `runAll/scripts/conf_loader.py` |
| `load_app_config(app_dir)` | 加载单个 app 配置 | 移到 `runAll/scripts/conf_loader.py` |
| `build_runtime_snapshot()` | 构建全量快照 | 移到 `runAll/scripts/conf_loader.py` |
| `load_domain_events_config()` | 加载领域事件配置 | 移到 `runAll/scripts/conf_loader.py` |
| `load_git_oauth_catalog()` | 加载 Git OAuth 目录 | 移到 `runAll/scripts/conf_loader.py` |

新模块 `runAll/scripts/conf_loader.py` 将：
1. 包含上述所有函数（从 task2app 版本提取，消除重复）
2. 复用 `conf_lib.py` 已有的 `load_yaml`、`deep_merge`、`repo_root`
3. 不依赖 task2app 的任何模块

**为什么可以这样做**：`task2app/Saas_project/config/conf_loader.py` 的原始职能是让 task2app 内部代码读取 `conf/` 配置。但如果 runAll 是配置管理的 owner，那 runAll 应该有自己的加载器。task2app 的 conf_loader 可以之后改为委托 runAll 的版本，但那是另一次重构的范围。

### 路径变更清单

| # | 调用方 | 当前引用 | 新引用 |
|---|--------|---------|--------|
| 1 | `taskSSE/src/config.mjs:67` | `scripts/conf-read.py` | `runAll/scripts/conf-read.py` |
| 2 | `taskSSE/src/config.mjs:78` | `scripts/conf-read.py` | `runAll/scripts/conf-read.py` |
| 3 | `conf/core/django/sync.sh` | `$ROOT/scripts/conf-sync.py` | `$ROOT/runAll/scripts/conf-sync.py` |
| 4 | `conf/frontend/vue/sync.sh` | 同上 | 同上 |
| 5 | `conf/auth/task-auth/sync.sh` | 同上 | 同上 |
| 6 | `conf/auth/git-oauth/sync.sh` | 同上 | 同上 |
| 7 | `conf/events/domain-events/sync.sh` | 同上 | 同上 |
| 8 | `conf/ai/ai-provider/sync.sh` | 同上 | 同上 |
| 9 | `conf/gateway/task-sse/sync.sh` | 同上 | 同上 |
| 10 | `runAll/src/conf_sync.go:31` | `filepath.Join(root, "scripts", "conf-sync-all.sh")` | `filepath.Join(root, "runAll", "scripts", "conf-sync-all.sh")` |
| 11 | `runAll/src/conf_sync.go:39` | `bash scripts/conf-sync-all.sh` | `bash runAll/scripts/conf-sync-all.sh` |
| 12 | `scripts/ci/check_conf_sync.sh:33` | `./scripts/conf-sync-all.sh` | `./runAll/scripts/conf-sync-all.sh` |
| 13 | `conf-sync-all.sh:9` | `python3 "$ROOT/scripts/conf-sync.py"` | `python3 "$ROOT/runAll/scripts/conf-sync.py"` |
| 14 | `conf/README.md` | `scripts/conf-read.py` | `runAll/scripts/conf-read.py` |

### conf_lib.py 的 repo_root 调整

```python
# 当前（scripts/ 下）
def repo_root() -> Path:
    here = Path(__file__).resolve().parent.parent  # scripts/ → parent = monorepo root ✓

# 迁移后（runAll/scripts/ 下）
def repo_root() -> Path:
    here = Path(__file__).resolve().parent.parent  # runAll/scripts/ → parent = runAll/
                                                    # parent.parent = monorepo root ✓
```

路径层级不变——`runAll/` 也在 monorepo 根下，所以 `parent.parent` 仍然正确。

### 不处理的内容

- `docs/superpowers/plans/` 中的历史文档引用：不更新（历史记录）
- `task2app/Saas_project/config/conf_loader.py`：保留不动（task2app 内部仍在使用）
- `scripts/ci/check_ddd_bdd_compliance.py` 中对 `check_conf_sync.sh` 的调用：check_conf_sync.sh 保留在 `scripts/ci/` 下，仅更新其内部路径

---

## 四、执行步骤

### Step 1：创建 runAll/scripts/conf_loader.py
- 从 `task2app/Saas_project/config/conf_loader.py` 提取所需函数
- 复用 `conf_lib.py` 的 `load_yaml`、`deep_merge`、`repo_root`
- 自包含，无 task2app 依赖

### Step 2：迁移脚本文件
- 移动 `conf-read.py`、`conf-sync.py`、`conf_lib.py`、`conf-sync-all.sh`、`migrate_port_config_to_conf.py` 到 `runAll/scripts/`
- 更新 `conf-read.py` 的 import：从 `config.conf_loader` → 从本地 `conf_loader`
- 更新 `conf-sync.py` 的 import：`from conf_lib` 路径不变（同一目录）

### Step 3：更新所有路径引用（14 处）
- 按上表逐文件修改

### Step 4：验证
- 运行 `python3 runAll/scripts/conf-read.py snapshot-json` 确认输出一致
- 运行 `bash runAll/scripts/conf-sync-all.sh` 确认所有 conf 子目录同步成功
- 运行 `bash scripts/ci/check_conf_sync.sh` 确认 CI 检查通过
- 运行 `cd runAll && go test ./...` 确认 Go 测试通过
- 手动验证 taskSSE 启动：`node taskSSE/src/index.mjs` 能读取配置

### Step 5：清理
- 删除 `scripts/conf-read.py`、`scripts/conf-sync.py`、`scripts/conf_lib.py`、`scripts/conf-sync-all.sh`、`scripts/migrate_port_config_to_conf.py`
- 删除 `scripts/__pycache__/`

---

## 五、风险与缓解

| 风险 | 缓解 |
|------|------|
| conf_loader 逻辑迁移后有差异 | 迁移后对比 `conf-read.py snapshot-json` 输出，确保字节级一致 |
| 遗漏某个调用方 | Step 3 前先 `grep -rn 'scripts/conf-read\|scripts/conf-sync'` 全仓扫描，确保无遗漏 |
| taskSSE 在 CI 中找不到新路径 | CI 环境已有 `runAll/` 目录，路径天然存在 |
| conf_lib 和 task2app 的 conf_loader 持续分化 | 短期可接受（不同 owner），长期 task2app 可委托 runAll |

---

## 六、价值流影响

本次变更涉及以下已有价值流步骤：

- **配置同步流**（conf/ 全域）：`sync.sh` 文件的脚本路径变更，功能行为不变
- **runAll 编排流**：`conf_sync.go` 路径更新，运行时行为不变
- **taskSSE 配置加载流**：`config.mjs` 路径更新，运行时行为不变

不涉及新增价值流或字段变更。

---

## 七、领域概念清单

为后续 `/5-ddd-领域设计驱动` 准备的轻量概念：

- **Bounded Context**: `配置管理 (Configuration Management)` — runAll 是此上下文的 owner
- **Key Entities**: `AppConfig`（每个 conf/<app>/ 子目录对应一个聚合）
- **Candidate Aggregates**: `ConfApp` — 以 app 目录为聚合根，config.yaml 为根实体，派生 YAML 为值对象
- **Domain Events**: `ConfigSynced` — 当 sync 操作完成后触发，通知消费方（如 taskSSE 热重载）

---

## 八、决策总结

| 决策点 | 选择 | 理由 |
|--------|------|------|
| conf-read/sync 归属 | runAll | runAll 是编排器，配置是编排的基础 |
| task2app conf_loader 依赖 | 提取到 runAll/scripts/conf_loader.py | 消除反向依赖，自包含 |
| Go 重写 vs Python 迁移 | Python 迁移 | 调用方（Node/taskSSE、shell/sync.sh）需要跨语言 CLI 接口；Go 重写会产生额外适配成本 |
| 向后兼容 | 不做兼容层 | 版本 ≤1.0.0，按项目规则禁止降级/兼容层 |
