# v7 Enterprise Landscape — Mermaid Diagram

> 架构版本: v7 🎯 target | 作者: claude | 日期: 2026-07-06 17:10
> 迭代: 任务域 Go 真源 — Django Todo 零残留

```mermaid
graph TD;
  gw["API Gateway (APISIX)"];
  django["Django SaaS [MODIFIED v7]\naccounts + company + todos"];
  authSvc["taskAuth (Go :8003)"];
  tps["taskProjectService (Go :8016)\n[MODIFIED v7] SQLite 真源"];
  tts["taskTaskService (Go :8017) planned"];
  tcs["taskCloudService (Go :8018) planned"];
  projectAPI["Project Management API"];
  taskAPI["Task Management API"];
  cloudAPI["Cloud Resource API"];
  projMgmt["Project Management Service"];
  sqlite["SQLite task_project.db"];
  postgres["PostgreSQL"];
  platV6["Plateau v6 — 3-Service design"];
  platV7["Plateau v8 — Phase 2 Task Domain Delivered"];
  gapProj["Gap: Project in Django ✅ closed"];
  gapRem["Gap: Phase 1b endpoints"];
  gapTask["Gap: Task in Django"];
  gapCloud["Gap: Aliyun in Django"];
  wp1["WP Phase 1 ✅ taskProjectService"];
  wp1b["WP Phase 1b — API parity"];
  wp2["WP Phase 2 — taskTaskService"];
  wp3["WP Phase 3 — taskCloudService"];
  tps --> projectAPI;
  projectAPI --> projMgmt;
  sqlite --> tps;
  tps --> authSvc;
  tps --> django;
  django --> postgres;
  wp1 --> gapProj;
  wp1 --> platV7;
  wp1b --> gapRem;
  wp2 --> gapTask;
  wp3 --> gapCloud;
  platV7 --> tps;
  platV6 --> tts;
  platV6 --> tcs;
```

## v7 变更要点

| 标记 | 内容 |
|------|------|
| 🟡 tps | SQLite 唯一真源，无 Django fallback |
| 🟡 django | projects_* 表与公网 CRUD 已删，Todo 保留 loose workspace_id |
| 🟢 sqlite | 项目域数据存储从 PostgreSQL 迁至 SQLite 文件 |
| ✅ gapProj | WP Phase 1 已关闭（502 修复 + 切流 + migration 0051） |
| ⏳ gapRem | Phase 1b：switch-workspace、gitlab-sync 等待补 |

## 架构变迁 v6→v7

```
Plateau v6 (design) ──WP1──▶ Gap: Project in Django (closed)
                              ──▶ Plateau v8 (Phase 1 live)
                              ──WP1b──▶ Gap: Phase 1b endpoints
Plateau v6 ──WP2/3──▶ Gap: Task / Cloud (open)
```
