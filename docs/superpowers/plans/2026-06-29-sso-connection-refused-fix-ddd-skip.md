# DDD 跳过声明: SSO Bridge Connectivity Fix

> 输入: docs/specs/sso-connection-refused-fix/design.md

## 跳过理由

本次变更符合 DDD 步骤的跳过条件：

- **配置变更**: `conf/ai/ai-provider/config.yaml` host + allowedExtendHosts
- **路径修正**: `port_config_loader.py` / `run.sh` 路径对齐
- **无新增领域概念**: 不引入新的 Entity、Value Object、Aggregate、Domain Event
- **已有领域模型不变**: SSO Bridge JWT 签发/换票机制、PlatformStaff/Vendor 聚合均不变

## 已有领域概念（来自设计文档，供参考）

| 类别 | 概念 |
|------|------|
| Bounded Context | ai-provider、auth |
| Key Entities | PlatformStaff、User、Vendor |
| Domain Events | StaffBridgeTokenIssued、SSOTokenExchanged |
