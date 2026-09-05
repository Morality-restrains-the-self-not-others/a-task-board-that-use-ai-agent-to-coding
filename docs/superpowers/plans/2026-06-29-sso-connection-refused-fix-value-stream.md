# Value Stream: SSO Bridge Connectivity Fix

> Derived from design: `docs/specs/sso-connection-refused-fix/design.md`

## Value Summary

管理员从主站 system-admin 通过 SSO 桥接跳转到镜像市场管理后台（Saas_Ai_Provider :8010），不再出现 Connection Refused，SSO 换票流程完整可用。

## Related Value Streams

Greenfield — no existing value streams for this topic area.

## End-to-End Flow

```
管理员点击「镜像市场管理（SSO）」 
  → 主站签发 staff_bridge JWT (302 redirect) 
  → 浏览器跳转 ai-provider :8010/admin#sso_bridge=<jwt> 
  → ai-provider SPA 提取 hash 中的 bridge token 
  → POST /api/auth/sso/exchange/ 换票 
  → 获取本地 access token → 管理员已登录镜像市场
```

## Value Increments

### Increment 1: 端口可达性修复 (Thin Slice)
**Value to user:** 点击 SSO 链接后浏览器能成功连接到 ai-provider，不再显示 Connection Refused
**Scope:** 
- `conf/ai/ai-provider/config.yaml`: host 改为 0.0.0.0，添加 allowedExtendHosts
- `port_config_loader.py` + `run.sh`: 修复配置路径 conf/ai-provider/ → conf/ai/ai-provider/
- `settings.py`: dev 模式 ALLOWED_HOSTS = ["*"]
**Depends on:** nothing

### Increment 2: E2E 验证
**Value to user:** 自动化测试持续验证 SSO 桥接可用性
**Scope:** Playwright E2E 测试覆盖完整 SSO 流程
**Depends on:** Increment 1
