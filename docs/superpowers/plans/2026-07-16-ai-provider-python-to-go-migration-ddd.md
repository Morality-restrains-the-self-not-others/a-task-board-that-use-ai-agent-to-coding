# DDD 领域模型：ai-provider（taskAiProvider）

**日期：** 2026-07-16  
**限界上下文：** AI Provider Marketplace  

## 聚合根

| 聚合 | 根实体 | 不变量 |
|------|--------|--------|
| VendorAccount | Vendor | email 唯一；saas_user_id 至多绑一主站用户 |
| StaffAccount | PlatformStaff | username 唯一；saas_superadmin_id 唯一 |
| ImageGroup | ContainerImageGroup | 属唯一 Vendor；内含多版本 |
| ContainerImageVersion | VendorContainerImage | (group, version) 唯一；状态机见下 |
| CloudServerImage | VendorCloudServerImage | 属 Vendor；可选 UserDataTemplate |
| UserDataTemplate | UserDataTemplate | Staff 管理；Vendor 只读 list |

## 实体 / 值对象

- **Association** (ContainerCloudServerAssociation)：container×platform×region 唯一
- **ReviewHistory**：审批动作记录（非聚合根，归属 ContainerImage）
- **CloudServerImageUserData**：1:1 云镜像 userdata + verification_secret
- **OIDCAuthSession** (VO)：state/PKCE verifier，短生命周期
- **AccessTokenClaims** (VO)：iss/sub/typ/exp

## 状态机（VendorContainerImage）

```
draft --submit--> pending_review --approve--> approved
                 |                \--reject--> rejected
                 \--withdraw--> draft
approved --unpublish--> draft (staff)
rejected --(edit)--> draft
```

## 领域服务

- `OidcAuthenticationService`：换票 + 匹配 Vendor/Staff
- `SsoBridgeExchangeService`：bridge JWT → local JWT
- `RegistryManifestService`：解析架构/size
- `ReviewService`：Approve/Reject + 写 History

## 仓储接口

- VendorRepository, StaffRepository, ImageGroupRepository, ContainerImageRepository, CloudServerImageRepository, UserDataTemplateRepository, AssociationRepository

## 领域事件

见设计文档 §6；首期 LoggingPublisher。
