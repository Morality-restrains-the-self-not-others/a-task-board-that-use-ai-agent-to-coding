# v69 Application Integration — 移除本机模拟启动 (Target, 2026-08-09 22:25)

```mermaid
graph TD;
  taskFE["taskFE (Vue SPA) [MODIFIED v69: 「模拟启动」面板删除]"];
  gateway["APISIX Gateway (:18081) [MODIFIED v69: mock-run-container 代理面 (start/stop/status/env-defaults) 删除]"];
  taskCloud["taskCloudService (Go :8018) [MODIFIED v69: /api/internal/mock-run/* 删除 + comment_csc_bootstrap mock/空平台报错拦截 + reconcile/provision 移除 mock-run]"];
  mockRunProxy["🔴 mock-run-container 代理面 (start/stop/status/env-defaults) [DEPRECATED v69: 删除]"];
  mockRunInternalAPI["🔴 /api/internal/mock-run/* (env-defaults / resolve-image) [DEPRECATED v69: 删除]"];
  mockRunSidecar["🔴 mock_run_container (Python Flask :8796) [DEPRECATED v69: 本机 Docker 模拟运行器 — 删除]"];
  goRunSidecar["🔴 go_run_container (Go :8796) [DEPRECATED v69: mock_run_container Go 重写 — 删除]"];
  plateauV68["Plateau v68 — 插件 OIDC 白名单管理"];
  plateauV69["Plateau v69 — 移除本机模拟启动"];
  gapMock["Gap: 本机容器模拟启动与云端启动双轨并存 → 启动行为不统一、本机 Docker 依赖"];
  wpMock["WP-remove-local-mock-start (P1 删除运行器+编排 → P2 gateway/cloud 删除转发 → P3 FE 面板删除 → P4 comment bootstrap 报错拦截 → P5 测试清理+验证)"];
  taskFE --> mockRunSidecar;
  gateway --> mockRunProxy;
  mockRunProxy --> mockRunSidecar;
  mockRunProxy --> mockRunInternalAPI;
  mockRunInternalAPI --> taskCloud;
  mockRunSidecar --o goRunSidecar;
  plateauV68 --> gapMock;
  wpMock --|> gapMock;
  wpMock --|> plateauV69;
```
