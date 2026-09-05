# v85 application-integration — pluggable-multi-region-gitservice

保留现网 GitLab CE（`gitlab.${baseDomain}` / `tencent-shanghai-5`）。  
上海机新增独立实例 `tencent-sh-1`（`gitlab-tencent-sh-1.${baseDomain}`，HTTP :8014，SSH :2223）。  
`billing_gitlab_region` 为可插拔注册表；租户无平台默认，须显式选购；开通按 region 调对应 Admin API。

```mermaid
flowchart LR
  FE["taskFE MOD"] -->|选购 region| Bill["taskBill MOD"]
  Bill -->|读路由| Reg["billing_gitlab_region"]
  Bill -->|写配额| Quota["tenant_gitlab_resource"]
  Bill -->|Admin API A| GL1["GitLab 现网 :8012"]
  Bill -->|Admin API sh-1 NEW| GL2["GitLab SH-1 :8014"]
  Edge["SH nginx MOD"] --> GL1
  Edge -->|gitlab-tencent-sh-1| GL2
  Auth["taskAuth MOD"] -->|OIDC| GL1
  Auth -->|独立 client| GL2
```
