# 测试意图：管理端赠送 GitLab 资源须指定区域

## 覆盖

| 场景 | 测试 | 期望 |
|------|------|------|
| 磁盘缺 region | `taskBill/src/admin_grant_region_test.go` `TestAdminGrantGitlabDiskRequiresRegion` | error 含 `region required` |
| 非法 slug | `TestAdminGrantGitlabDiskUnknownRegion` | error 含 `region not found` |
| 指定区域落库 | `TestAdminGrantGitlabDiskWritesSelectedRegion` | `disk_gb` 落在所选 region；订单行 `region` 匹配 |
| 两区域不串 | `TestAdminGrantGitlabDiskRegionsIndependent` | 两行独立累加 |
| 流量同样必填 | `TestAdminGrantGitlabTrafficRequiresRegion` | 缺 region 失败 |
| 任务帖不需 region | `TestAdminGrantTaskPostWithoutRegionStillWorks` | 成功 |
| 前端展示与 POST | `taskFE/app/src/views/SystemAdminGrantPoints.region.unit.test.js` | GitLab 类型出现下拉；body 含 region |

## 命令

```bash
cd /tmp/ram-work/taskBill && go test ./src -count=1 -run 'AdminGrant'
cd /tmp/ram-work/taskFE/app && npx vitest run src/views/SystemAdminGrantPoints.region.unit.test.js
```
