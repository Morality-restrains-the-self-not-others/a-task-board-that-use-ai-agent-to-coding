# 测试意图：Navbar「代码仓库」区域下拉

| ID | 场景 | 期望 |
|----|------|------|
| TI-NGS-01 | GET gitlab-resources 无 region，租户已获赠一区 | 200，`resources[]` 含该区且有 `gitlab_web_url` |
| TI-NGS-02 | GET gitlab-resources 无 region，空租户 | 200，`resources: []`（非 null） |
| TI-NGS-03 | GET gitlab-resources?region=slug | 200，单区详情字段（既有契约） |
| TI-NGS-04 | Navbar 单区资源 | 触发器为 button；mouseenter 后菜单 1 项外链 |
| TI-NGS-05 | Navbar 多区资源 | click 展开；菜单项数 = jumpable 数 |
| TI-NGS-06 | Navbar 无资源 ready | `a[data-testid=nav-git-service]` href=`/pricing/` |
| TI-NGS-06b | Navbar 无资源 error/unknown | href=当前页（fail-open） |
| TI-NGS-07 | 悬停打开后 mouseleave | 菜单收起 |

## 对应实现测例

- `taskBill/src/gitlab_convention_path_http_test.go`：`TestHandleGitlabResources_WithoutRegion*`
- `taskFE/app/src/components/Navbar.ui.test.js`：代码仓库按仓库资源跳转
