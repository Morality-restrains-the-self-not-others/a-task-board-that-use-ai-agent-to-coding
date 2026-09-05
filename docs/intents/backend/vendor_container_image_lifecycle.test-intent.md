# 测试意图：厂商撤回/下架/删除容器镜像

## 测试目标

领域守卫与 HTTP 契约与事件投递。

## 用例矩阵

| # | 路径 | 期望 |
|---|------|------|
| D1 | VendorWithdraw(approved) | draft && !IsActive |
| D2 | DeleteGuard(active) | error |
| D3 | DELETE active | 400 |
| D4 | DELETE approved inactive | 204 + ContainerImageDeleted |
| D5 | POST withdraw approved | draft + ContainerImageUnpublished |

可执行：`go test ./domain ./src -count=1 -run 'CanDelete|VendorWithdraw|VendorContainerLifecycle'`
