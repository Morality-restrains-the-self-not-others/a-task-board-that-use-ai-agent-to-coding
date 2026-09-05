# 测试意图：厂商门户版本行管理手段

## 测试目标

证明已上架版本行具备可见管理按钮，且下架/删除与激活守卫符合领域规则。

## 测试分层

- 前端单测：操作矩阵、CSS 栅格、Vue 接线、clickGuard。
- 领域单测：CanDelete / VendorWithdraw。
- HTTP 单测：withdraw approved、DELETE 守卫、事件名。

## 用例矩阵

| # | 给定 | 当 | 则 |
|---|------|----|----|
| 1 | approved && !is_active | versionRowActions | 含 runtimeEnv, activate, withdraw, delete |
| 2 | approved && is_active | versionRowActions | 含 withdraw，不含 delete |
| 3 | draft | versionRowActions | 含 edit, delete, submit |
| 4 | CSS | 读 VendorPortal.css | version-row 5 列且 .version-url grid-column 1/-1 |
| 5 | 激活镜像 | DELETE | 400 含「激活」 |
| 6 | 非激活 approved | DELETE | 204 + ContainerImageDeleted |
| 7 | approved | POST withdraw | 200 status=draft + Unpublished 事件 |

## 通过标准

上述单测全绿。
