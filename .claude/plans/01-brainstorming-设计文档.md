# 设计文档：修复 aiProvider Vite 启动 ENOSPC 崩溃

## 1. 问题描述

**日志**：
```
VITE v5.4.21  ready in 705 ms
  ➜  Local:   http://127.0.0.1:5173/
Error: ENOSPC: System limit for number of file watchers reached, watch '.../src/router'
    at FSWatcher.<computed> (node:internal/fs/watchers:254:19)
  errno: -28,
  syscall: 'watch',
  code: 'ENOSPC',
```

**现象**：Vite 开发服务器启动成功（端口绑定 + ready），但在初始化文件监听时因 `ENOSPC` 崩溃。run.sh 的 `start_vite_dev()` 检测到端口 LISTEN 后返回 success，但 Vite 随后立即退出。

## 2. 根因分析

### 2.1 直接原因：inotify 监听数达系统上限

```
当前限制: fs.inotify.max_user_watches = 65536
Vite 行为: 启动时为 HMR 注册大量文件监听（node_modules、src、dist 等目录）
结果:     监听数耗尽 → ENOSPC(errno=-28) → Vite 进程退出
```

Vite v5.4.21 启动流程：
```
1. Vite 进程启动 → PID 写入 vite.pid
2. 绑定 TCP 5173 → lsof 看到 LISTEN
3. "ready in 705 ms" 输出到 stdout
4. 注册文件监听 → ENOSPC 崩溃 → 进程退出
```

### 2.2 辅助原因：`start_vite_dev()` 競态检测

**文件**: `Saas_Ai_Provider/run.sh:158-169`

```bash
for _wait in {1..40}; do
    sleep 0.5
    if lsof -nP -iTCP:"$vite_port" -sTCP:LISTEN > /dev/null 2>&1; then
        _ok=1; break     # ← Vite 在第 3 步就绑定端口，此时检测到 LISTEN → 返回成功
    fi
    if ! ps -p "$vid" > /dev/null 2>&1; then
        break              # ← 但在返回前，Vite 尚未崩溃（崩溃发生在 ready 之后）
    fi
done
```

**时序竞态**：
```
T0:  Vite 启动 → PID 存在
T2:  Vite bind(5173) → TCP LISTEN
T3:  poll 检测到 LISTEN → _ok=1 → 返回成功 ✅
T4:  Vite 打印 "ready"，开始注册文件监听
T6:  Vite 注册 src/router 监听 → ENOSPC → 进程退出 💥
```

poll 循环在 T3 就退出了，T6 的崩溃未被检测到。虽然 `start_vite_dev` 返回 success，Vite 实际已死亡。

### 2.3 为什么 65536 不够

这是一个 monorepo，运行中的服务包括：
- 多个 Node.js 进程（前端 Vite dev server）
- Django 开发服务器
- 可能有 IDE（VS Code 也使用 inotify）
- GitLab 等容器服务也会消耗 inotify 监听

Vite 启动时为 `node_modules/`（即使部分忽略）、`src/`、`dist/` 的每个子目录注册监听，瞬时可达数千个。多服务并发时累计超过 65536。

---

## 3. 修复方案

### 修复 1（根因）: 提高 inotify 监听上限

**修改**: 系统级 sysctl 参数

```bash
# 当前
fs.inotify.max_user_watches = 65536

# 推荐
fs.inotify.max_user_watches = 524288
```

**执行方式**：
- 临时生效: `sudo sysctl -w fs.inotify.max_user_watches=524288`
- 永久生效: 写入 `/etc/sysctl.d/99-inotify.conf`

**判断理由**: 524288 是 Docker/Node.js monorepo 社区推荐值。每个监听只占用约 1KB 内核内存，524288 个监听约 512MB — 对开发机可接受。

### 修复 2（防御）: `start_vite_dev()` 增加启动后存活确认

**文件**: `Saas_Ai_Provider/run.sh:146-178`，`start_vite_dev()` 函数

在 port LISTEN 检测通过后，增加 2 秒延迟 + PID 存活二次确认：

```bash
if [ "$_ok" -eq 1 ]; then
    # 二次确认：延迟后检查 PID 是否仍存活（防止 ENOSPC 等启动后崩溃）
    sleep 2
    if ! ps -p "$vid" > /dev/null 2>&1; then
        echo -e "${RED}Vite 进程在启动后退出，请查看：$LOG_DIR/vite.log${NC}"
        rm -f "$VITE_PID_FILE"
        return 1
    fi
    echo -e "${GREEN}Vite 已监听 http://127.0.0.1:${vite_port}/ ...${NC}"
    return 0
fi
```

### 修复 3（预检）: 启动前检查 inotify 余量

**文件**: `Saas_Ai_Provider/run.sh`，在 `start_vite_dev()` 开头增加

```bash
# 检查 inotify 监听余量，低于 20% 时警告
local _max_watches _used _avail
_max_watches=$(cat /proc/sys/fs/inotify/max_user_watches 2>/dev/null || echo 0)
if [ "$_max_watches" -gt 0 ] 2>/dev/null; then
    _used=$(lsof -nP 2>/dev/null | grep -c inotify || echo 0)
    _avail=$((_max_watches - _used))
    if [ "$_avail" -lt $((_max_watches / 5)) ]; then
        echo -e "${YELLOW}inotify 监听余量不足（已用 $_used / $_max_watches），建议提高 fs.inotify.max_user_watches${NC}"
    fi
fi
```

### 修复 4（Vite 配置优化）: 明确排除不需要监听的目录

**文件**: `Saas_Ai_Provider/frontend/vite.config.js`

Vite 5 默认排除 `node_modules`，但可显式配置确保：

```js
server: {
    watch: {
        ignored: ['**/node_modules/**', '**/.git/**', '**/dist/**'],
    },
    // ... 现有配置不变
}
```

---

## 4. 改动范围

| 文件 | 改动类型 | 估计行数 |
|------|---------|---------|
| 系统 `/etc/sysctl.d/99-inotify.conf` | 新增 sysctl 配置 | 1 行 |
| `Saas_Ai_Provider/run.sh` — `start_vite_dev()` | 存活二次确认 | ~6 行 |
| `Saas_Ai_Provider/run.sh` — `start_vite_dev()` | inotify 余量预检 | ~8 行 |
| `Saas_Ai_Provider/frontend/vite.config.js` | watch.ignored 显式配置 | ~3 行 |

---

## 5. Domain Concept Inventory

不适用 — 纯基础设施/运维修复，无业务领域概念。

---

## 6. Value Stream Impact

受影响的价值流：**platform-dev-database-reset** 和 **platform-centralized-logging**（平台与本地开发）

| 步骤 | 影响 |
|------|------|
| N/A — 无现有步骤覆盖 Vite 启动可靠性 | 可选：新增 `vite-startup-reliability` 步骤到 `dev-tools-log-viewer` 或新建 stream |

实际上本次修复是平台基础设施层面的改进，不需要新增价值流条目。

---

## 7. 实施步骤

1. **系统级**: 提高 `fs.inotify.max_user_watches` 至 524288
2. **run.sh**: `start_vite_dev()` 增加存活二次确认（port LISTEN 后 sleep 2 + PID 检查）
3. **run.sh**: `start_vite_dev()` 增加 inotify 余量预检
4. **vite.config.js**: 显式配置 `server.watch.ignored`
5. **验证**: 重启 aiProvider，确认 Vite 不再 ENOSPC 崩溃

---

## 8. 风险评估

- **极低风险**：所有改动为防御性增强，不改变正常启动流程
- **sysctl 改动**: 需要 sudo 权限；524288 是社区验证安全值
- **回滚**: 直接 revert 即可

---

## 总结清单

- **根因修复**: 方案 1（提高 max_user_watches 至 524288，推荐）、方案 2（仅 Vite 配置优化不提高系统限制）
- **竞态修复**: 方案 1（存活二次确认 sleep 2 + PID 检查，推荐）、方案 2（仅依赖现有的 PID 检查逻辑）
- **预检**: 方案 1（启动前检查 inotify 余量，推荐）、方案 2（跳过预检，仅依赖修复 1+2）
