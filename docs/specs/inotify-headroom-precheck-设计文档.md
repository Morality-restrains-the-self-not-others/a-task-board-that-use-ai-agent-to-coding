# 设计文档：inotify 余量不足预检与系统性修复

## 1. 问题描述

### 1.1 触发场景

启动 `Saas_Ai_Provider` 的 Vite 开发服务器时，出现 **inotify 余量不足预检警告**。Vite 虽然在启动阶段成功绑定了端口（`ready in 705 ms`），但在注册文件监听时因 `ENOSPC`（`errno=-28`）崩溃。

### 1.2 实际日志证据

**文件**: `task2app/Saas_Ai_Provider/logs/vite.log:13`

```
Error: ENOSPC: System limit for number of file watchers reached,
       watch '/tmp/ram-work/task2app/Saas_Ai_Provider/frontend/src/router'
    at FSWatcher.<computed> (node:internal/fs/watchers:254:19)
    errno: -28,  syscall: 'watch',  code: 'ENOSPC',
```

**时序竞态**:
```
T0:  Vite 启动 → PID 存在
T2:  Vite bind(5173) → TCP LISTEN
T3:  run.sh poll 检测到 LISTEN → _ok=1 → 返回成功 ✅
T4:  Vite 打印 "ready"，开始注册文件监听
T6:  Vite 注册 src/router 监听 → ENOSPC → 进程退出 💥
```

poll 循环在 T3 就退出了，T6 的崩溃未被检测到。虽然 `start_vite_dev` 返回 success，Vite 实际已死亡。

---

## 2. 根因分析

### 2.1 直接原因：inotify 监听数达系统上限

```
当前限制: fs.inotify.max_user_watches = 65536 (系统默认值)
当前限制: fs.inotify.max_user_instances = 128
Vite 行为: 启动时为 HMR 注册文件监听（src/ 每个子目录一个 watch）
结果:     监听数耗尽 → ENOSPC → Vite 进程退出
```

### 2.2 ⭐ 主要隐藏消费者：ramsync-daemon.sh

**这是之前分析未识别出的关键根因。**

`/home/ljy/bin/ramsync-daemon.sh`（自 6月13日起持续运行）使用 `inotifywait -r` 递归监听整个 `/tmp/ram-work` 目录树：

```bash
# ramsync-daemon.sh:59
inotifywait -t 60 -e modify,create,delete,move "$RAM_SOURCE" -r --format '%w%f' 2>/dev/null
```

**inotify 监听消耗估算**:

| 类别 | 目录数 | 说明 |
|------|--------|------|
| `/tmp/ram-work` 全量目录 | 38,696 | 递归 -r 不加 exclude |
| 排除 .git 后 | 36,325 | inotifywait 在 kernel 层仍注册 watch |
| 排除 .git + node_modules 后 | 34,431 | EXCLUDE_ARGS 仅用于 rsync，不用于 inotifywait |
| **ramsync-daemon 实际消耗** | **~34,000–38,696** | `--exclude` 只过滤输出不减少 watch 注册 |

**结论**: ramsync-daemon 单独消耗了 **~52%–59%** 的 inotify watch 配额（34,000 / 65,536）。

加上其他消费者：
- VS Code / Cursor IDE: ~5,000–10,000 watches
- Docker 容器（GitLab CE, Kafka, Redis 等）: 每个容器 ~500–2,000 watches
- Django `runserver` auto-reload: ~500 watches
- GitLab CE 容器内 `app/assets` 目录树: 1,982 dirs

**累计估算**: 34,000 + 10,000 + 5,000 + 500 + 2,000 ≈ **51,500 watches**，已接近 65,536 上限。

当 Vite 再尝试注册 100–500 个新 watch 时，立即触发 ENOSPC。

### 2.3 辅助原因：`start_vite_dev()` 端口检测竞态

**文件**: `task2app/Saas_Ai_Provider/run.sh:158-169`

```bash
for _wait in {1..40}; do
    sleep 0.5
    if lsof -nP -iTCP:"$vite_port" -sTCP:LISTEN > /dev/null 2>&1; then
        _ok=1; break     # ← Vite 在第 3 步就绑定端口，此时检测到 LISTEN → 返回成功
    fi
    if ! ps -p "$vid" > /dev/null 2>&1; then
        break
    fi
done
```

Vite v5 启动时序：bind → port LISTEN → "ready" → 注册文件监听 → ENOSPC。poll 在 port LISTEN 后就返回成功，后续的崩溃未被检测。

### 2.4 Vite 配置缺少显式排除

**文件**: `task2app/Saas_Ai_Provider/frontend/vite.config.js`

当前配置中 `server` 块**没有** `watch.ignored` 显式配置。Vite 5 虽然默认排除 `node_modules`，但在此 monorepo 环境中，Vite 可能遍历到工作区根目录的其他大型目录（如 `conf/`、`db/`、`gitlab-ce/` 等）。

对比：`front_project/app/vite.config.js` 使用了 `usePolling: true`，完全避免 inotify。

---

## 3. 修复方案

### 修复优先级总结

| 优先级 | 修复项 | 效果 | 复杂度 |
|--------|--------|------|--------|
| **P0** | ramsync-daemon 改用轮询替代 inotify | 释放 ~34,000 watches (50%+) | 低 |
| **P0** | 提高 `max_user_watches` 至 524288 | 提升 8× 余量 | 极低 |
| **P1** | `start_vite_dev()` 启动后存活确认 | 防止静默崩溃 | 低 |
| **P1** | `start_vite_dev()` inotify 余量预检 | 早发现、早警告 | 低 |
| **P2** | Vite `watch.ignored` 显式配置 | 减少单实例 watch 消耗 | 极低 |

---

### 修复 1（⭐ P0 根因）: ramsync-daemon 用轮询替代 inotify

**问题**: `inotifywait -r` 对整个 `/tmp/ram-work`（38,696 个目录）注册内核级监听，消耗 ~52% inotify 配额。

**方案**: 将 `ramsync-daemon.sh` 从 inotify 事件驱动改为定时轮询（如每 10 秒 rsync），消除 inotify 监听消耗。

**修改文件**: `/home/ljy/bin/ramsync-daemon.sh`

```bash
# 替换 monitor_loop() 中的 inotifywait 轮询：

monitor_loop() {
    log "Starting poll-based monitor loop (inotify-free, interval=${POLL_INTERVAL:-10}s)"
    while true; do
        sleep "${POLL_INTERVAL:-10}"
        do_sync
    done
}
```

**理由**:
- rsync 本身已做增量同步（只传输变更），10 秒间隔对 RAM→磁盘同步场景可接受
- `MIN_SYNC_INTERVAL=10` 已存在，轮询天然对齐此间隔
- 释放 ~34,000 inotify watches，从根本上解决问题
- 零额外依赖，改动仅 ~4 行

**风险**: 定时轮询的同步延迟 ≤10 秒（vs inotify 的实时）。对 RAM work 目录同步场景完全可接受。

---

### 修复 2（P0 防御）: 提高 inotify 系统上限

**修改**: 系统级 sysctl 参数

```bash
# 当前
fs.inotify.max_user_watches = 65536

# 推荐
fs.inotify.max_user_watches = 524288
```

**执行方式**:
- 临时生效: `sudo sysctl -w fs.inotify.max_user_watches=524288`
- 永久生效: 写入 `/etc/sysctl.d/99-inotify.conf`

```
# /etc/sysctl.d/99-inotify.conf
fs.inotify.max_user_watches = 524288
```

**判断理由**: 524288 是 Docker/Node.js monorepo 社区推荐值。每个监听占用约 1KB 内核内存，524288 ≈ 512MB — 对开发机可接受。即使 ramsync-daemon 修复后，IDE + Docker + Django 等仍可能消耗 20,000+ watches，需要足够的余量。

---

### 修复 3（P1 防御）: `start_vite_dev()` 启动后存活确认

**修改文件**: `task2app/Saas_Ai_Provider/run.sh:170-173`

在 port LISTEN 检测通过后，增加 2 秒延迟 + PID 存活二次确认：

```bash
if [ "$_ok" -eq 1 ]; then
    # 二次确认：延迟后检查 PID 是否仍存活（防止 ENOSPC 等启动后崩溃）
    sleep 2
    if ! ps -p "$vid" > /dev/null 2>&1; then
        echo -e "${RED}Vite 进程在启动后退出（可能是 inotify 耗尽、端口冲突等），请查看：$LOG_DIR/vite.log${NC}"
        rm -f "$VITE_PID_FILE"
        return 1
    fi
    echo -e "${GREEN}Vite 已监听 http://127.0.0.1:${vite_port}/ （修改 frontend/src 将热更新）${NC}"
    echo -e "${GREEN}API 请求经 Vite 代理至 Django（见 vite.config.js proxy）${NC}"
    return 0
fi
```

---

### 修复 4（P1 预检）: 启动前检查 inotify 余量

**修改文件**: `task2app/Saas_Ai_Provider/run.sh`，在 `start_vite_dev()` 函数中启动 Vite 之前插入：

```bash
# 检查 inotify 监听余量，低于 20% 时警告
local _max_watches _used _avail
_max_watches=$(cat /proc/sys/fs/inotify/max_user_watches 2>/dev/null || echo 0)
if [ "$_max_watches" -gt 0 ] 2>/dev/null; then
    _used=$(find /proc/[0-9]*/fd -lname 'inotify*' 2>/dev/null | wc -l)
    _avail=$((_max_watches - _used))
    if [ "$_avail" -lt $((_max_watches / 5)) ]; then
        echo -e "${YELLOW}⚠ inotify 监听余量不足（已用 ${_used} / ${_max_watches}，可用 ${_avail}），建议：${NC}"
        echo -e "${YELLOW}  1. 提高系统上限: sudo sysctl -w fs.inotify.max_user_watches=524288${NC}"
        echo -e "${YELLOW}  2. 检查 ramsync-daemon 是否占用过多: ps aux | grep ramsync${NC}"
    fi
fi
```

**注意**: 使用 `find /proc/*/fd -lname 'inotify*'` 比 `lsof | grep inotify` 更精确（后者会匹配进程名、库名中包含 inotify 的行，可能多计数）。

---

### 修复 5（P2 优化）: Vite 显式配置 watch.ignored

**修改文件**: `task2app/Saas_Ai_Provider/frontend/vite.config.js`

在 `server` 配置块中新增显式排除：

```js
server: {
    // ... 现有配置不变
    watch: {
        ignored: [
            '**/node_modules/**',
            '**/.git/**',
            '**/dist/**',
            '**/__pycache__/**',
            '**/*.pyc',
            '**/logs/**',
        ],
    },
    // ...
}
```

---

## 4. 改动范围

| 文件 | 改动类型 | 估计行数 |
|------|---------|---------|
| `/home/ljy/bin/ramsync-daemon.sh` | inotifywait → sleep 轮询 | ~4 行 |
| `/etc/sysctl.d/99-inotify.conf` | 新增 sysctl 持久化配置 | 1 行 |
| `task2app/Saas_Ai_Provider/run.sh` — `start_vite_dev()` | 存活二次确认 | ~6 行 |
| `task2app/Saas_Ai_Provider/run.sh` — `start_vite_dev()` | inotify 余量预检 | ~10 行 |
| `task2app/Saas_Ai_Provider/frontend/vite.config.js` | watch.ignored 显式配置 | ~8 行 |

---

## 5. Domain Concept Inventory

不适用 — 纯基础设施/运维修复，无业务领域概念。

---

## 6. Value Stream Impact

本修复属于平台基础设施层面（dev-tooling / 本地开发体验），不直接影响业务价值流。

受影响的价值流（间接）:
- **platform-dev-local-startup**（隐式）：所有依赖 Vite 热重载的本地开发流程受益
- **ai-provider-service**（隐式）：aiProvider 前端开发体验改善

实际上无需新增价值流条目，也不需要修改现有 value-stream.yaml。

---

## 7. 实施步骤

### 阶段 1：根因修复（立即见效）
1. **ramsync-daemon.sh**: 将 `inotifywait -r` 替换为 `sleep + rsync` 轮询
2. **验证**: `find /proc/*/fd -lname 'inotify*' | wc -l` 应大幅下降（预期从 ~34,000 → ~5,000）

### 阶段 2：系统防御（提高上限）
3. **sysctl**: 提高 `fs.inotify.max_user_watches` 至 524288
4. **持久化**: 写入 `/etc/sysctl.d/99-inotify.conf`

### 阶段 3：应用层防御
5. **run.sh**: `start_vite_dev()` 增加存活二次确认（port LISTEN 后 sleep 2 + PID 检查）
6. **run.sh**: `start_vite_dev()` 增加 inotify 余量预检
7. **vite.config.js**: 显式配置 `server.watch.ignored`

### 阶段 4：验证
8. 重启 aiProvider，确认 Vite 启动成功且无 ENOSPC 错误
9. 检查 vite.log 确认无 ENOSPC 崩溃
10. 验证 `inotify 余量预检` 在正常余量时**不**误报，在低余量时**正确**告警

---

## 8. 风险评估

- **极低风险**: 所有改动为防御性增强，不改变正常启动流程
- **ramsync-daemon 改动**: 从实时同步变为 10 秒轮询，对 RAM→磁盘备份场景影响可忽略
- **sysctl 改动**: 需要 sudo 权限；524288 是社区验证安全值
- **回滚**: 直接 revert 每项改动即可

---

## 总结清单

- **根因（新发现）**: ramsync-daemon.sh 用 `inotifywait -r` 监听整个 /tmp/ram-work（38,696 目录），消耗 ~52% inotify 配额 — 方案：改用 sleep 轮询
- **系统限制**: max_user_watches=65536 太低 — 方案：提高至 524288
- **竞态漏洞**: `start_vite_dev()` port LISTEN 后未做存活确认 — 方案：sleep 2 + PID 二次确认
- **预检缺失**: Vite 启动前不检查 inotify 余量 — 方案：启动前检查 /proc/sys + /proc/*/fd
- **Vite 配置**: 缺少显式 `watch.ignored` — 方案：显式排除 node_modules/.git/dist/logs
