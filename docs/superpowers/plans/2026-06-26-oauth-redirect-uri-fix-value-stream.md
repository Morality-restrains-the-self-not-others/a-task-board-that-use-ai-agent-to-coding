# Value Stream: Fix OAuth redirect_uri to Public Gateway

> Design: OAuth callback fails because redirect_uri uses `127.0.0.1:8443` (unreachable from user browser). Fix to `183.250.1.132:9443`.

## Value Summary

用户浏览器能正确到达 OAuth 回调地址，完成 GitLab 授权流程。

## End-to-End Flow

[GitLab 授权] → [302 到 redirect_uri] → [浏览器访问网关] → [APISIX 路由到 gitOauth] → [换 code 为 token] → [授权完成]

## Value Increments

### Increment 1: 修改 provider YAML + 同步 Doorkeeper (Thin Slice)
**Value:** OAuth 回调不再卡在 127.0.0.1:8443
**Scope:** 4 个 GitLab provider YAML redirect_uri → 重跑 sync 脚本
**Depends on:** nothing
