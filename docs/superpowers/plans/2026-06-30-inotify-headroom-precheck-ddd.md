# DDD 领域建模: inotify 余量不足预检与系统性修复

> 输入:
> - 设计文档: `docs/specs/inotify-headroom-precheck-设计文档.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-30-inotify-headroom-precheck-nfr-clarification.md`
>
> 输出使用者: `/7-plans-实施计划`

## 跳过声明

**本次变更不涉及领域建模。** 原因：

1. 纯基础设施/运维修复 — 所有改动在 shell 脚本、系统配置、Vite 配置层面
2. 无新增业务概念 — 不涉及实体、值对象、聚合、领域事件
3. 无新增限界上下文 — 不改变任何服务的业务边界
4. 无新增端口接口 — 不改动任何依赖反转架构

NFR 澄清文档已确认「领域模型影响: 无」。

## 改动文件清单（供 plans 步骤使用）

| 文件 | 类型 | 说明 |
|------|------|------|
| `/home/ljy/bin/ramsync-daemon.sh` | bash | inotifywait → sleep 轮询 |
| `/etc/sysctl.d/99-inotify.conf` | sysctl | max_user_watches=524288 |
| `task2app/Saas_Ai_Provider/run.sh` | bash | 存活确认 + 余量预检 |
| `task2app/Saas_Ai_Provider/frontend/vite.config.js` | js | watch.ignored |
