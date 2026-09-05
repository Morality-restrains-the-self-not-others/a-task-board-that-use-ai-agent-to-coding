# 价值流：测试角色 + GitLab 区域开发/发布模式

- **日期**: 2026-08-23
- **设计**: `docs/superpowers/specs/2026-08-23-tester-role-gitlab-region-access-mode-design.md`

## Related Value Streams

既有 GitLab 区域管理、用户列表列过滤。本流是**扩展**：账号多一个类别，区域多一个访问模式，目录/购买增加门禁。不撤销既有区域 CRUD。

## 价值增量（垂直切片）

| # | 增量 | 穿越栈 | 验收 |
|---|------|--------|------|
| V1 | 测试角色标志与筛选展示 | DDL → PATCH/列表 → FE 勾选/徽章/筛选 | role=tester；徽章「测试」 |
| V2 | 网关 X-User-Is-Tester | forward-auth → 头 | 测试账号为 1 |
| V3 | 区域 access_mode 管理 | DDL → 系统管理 GET/PUT/POST → FE 卡片 | 可设开发/发布 |
| V4 | 租户目录与购买门禁 | listGitlabRegions + purchase | 非测试不可见/不可买 development |

## 最小可交付

V1+V3+V4 必须同会话交付，否则开发区无使用主体或无门禁。V2 与 V4 耦合（无头则无法识别测试）。

## 测试点（同步 value-stream-test-integration.wsd）

- TP-tester-filter / TP-tester-badge / TP-tester-patch
- TP-forward-auth-tester-header
- TP-region-mode-save / TP-region-catalog-filter / TP-region-purchase-403
