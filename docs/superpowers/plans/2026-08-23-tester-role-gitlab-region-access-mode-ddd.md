# 领域模型：测试角色与区域访问模式

- **日期**: 2026-08-23
- **NFR**: `2026-08-23-tester-role-gitlab-region-access-mode-nfr-clarification.md`

## 限界上下文

- **Identity（taskAuth）**：账号标志 `is_tester` / `is_tenant`
- **Billing GitLab（taskBill）**：区域 `access_mode` 与配额购买

不新建服务。领域规则以纯函数落在各服务 `src/`（与现有包布局一致，不引入空 `domain/` 包以免与单体 main 冲突）。

## 值对象

```
GitlabRegionAccessMode = "release" | "development"
```

非法字符串拒绝。默认 `release`。

## 领域服务

```
CanUseGitlabRegion(mode GitlabRegionAccessMode, isTester bool) bool
  release -> true
  development -> isTester
```

## 实体

- AuthUser: +is_tester；AssignTester => is_tester=true, is_tenant=true
- GitlabRegion: +access_mode

## 领域事件

- UserTesterFlagChanged { user_id, is_tester, actor_user_id }
- GitlabRegionAccessModeChanged { region_slug, access_mode, actor_user_id }

幂等键与业务边界同粒度：user_id；region_slug。

## 端口

- UserFlagRepository（现有 db 函数）
- GitlabRegionRepository（现有 list/get/update）
- EventBus：既有 publishDomainEventKafka / publishEvent

无新消费者 intent。
