# v17 Enterprise Landscape — 移除本机模拟启动 (Target, 2026-08-09 22:25)

```mermaid
graph TD;
  feSvc["taskFE (Vue SPA) [MODIFIED v17: 「模拟启动」面板/mockRun tab/mockContainerStart flag 删除]"];
  containerGW["taskContainerGateway (Go :8014) [MODIFIED v17: mock-run-container 代理面删除]"];
  taskCloudSvc["taskCloudService (Go :8018) [MODIFIED v17: /api/internal/mock-run/* 删除 + comment_csc_bootstrap mock/空平台报错拦截]"];
  mockRunContainer["🔴 mock_run_container (Python Flask :8796) [DEPRECATED v17: 本机 Docker 模拟运行器 — 删除]"];
  goRunContainer["🔴 go_run_container (Go :8796) [DEPRECATED v17: mock_run_container Go 重写 — 删除]"];
  plateauV16["Plateau v16 — 插件 OIDC 白名单管理"];
  plateauV17["Plateau v17 — 移除本机模拟启动"];
  gapMock["Gap: 本机容器模拟启动与云端启动双轨并存 → 启动行为不统一、本机 Docker 依赖"];
  wpMock["WP-remove-local-mock-start (P1 删除运行器+编排 → P2 gateway/cloud 删除转发 → P3 FE 面板删除 → P4 comment bootstrap 报错拦截 → P5 测试清理+验证)"];
  feSvc --> mockRunContainer;
  containerGW --> mockRunContainer;
  taskCloudSvc --> mockRunContainer;
  mockRunContainer --o goRunContainer;
  plateauV16 --> gapMock;
  wpMock --|> gapMock;
  wpMock --|> plateauV17;
```
