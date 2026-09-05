# 价值流：评论启动日志 COS 归档

- **Date:** 2026-08-27
- **Design:** `docs/superpowers/specs/2026-08-27-startup-logs-cos-archive-design.md`

## 增量（唯一 MVP）

用户在工作台启动评论容器 → 启动日志出现在「启动日志」面板 → 刷新/释放后仍可看到完整时间线（MySQL 热表 + COS 归档）。

```
用户 @镜像 / 启动
  → Cloud 写 CCB 分片行（热）
  → 同事务外 best-effort 合并 COS bundle；Put 成功后删除分片行
  → SSE 仍推前端
  → 冷打开 list API 从 COS 拉取（分片仅补缺口）
  → 面板 h4「启动日志」渲染
```

## 测试点

| ID | 刺激 | 期望 |
|----|------|------|
| T1 | 默认 pathRule 渲染 | `workspace_*/task_*/comment_*/startup_logs.json` |
| T2 | pathRule 含 `..` | 拒绝 |
| T3 | insert 后 memory COS Get | bundle 含该 log id |
| T4 | 同一 id 再 persist | 不重复行 |
| T5 | COS Put 失败 | MySQL insert 仍成功且分片保留 |
| T6 | Put 成功后 list | 分片无该 id，从指针/COS 还原 |
| T7 | 非员工 GET COS 配置 | 403 |
| T8 | 员工 GET | 含 startupLogsPathRule，无 secretKey |

## 范围外

- 心跳行并入启动日志
- 克隆日志 COS
- 新前端面板 UI
