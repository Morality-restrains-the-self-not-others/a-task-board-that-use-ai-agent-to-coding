# [运行时] 保存 API 端点后 base_url 被拼接/覆写

## 基本信息
- 案例编号：FE-20260818-0101
- 录入日期：2026-08-18
- 最后更新：2026-08-18
- 录入人：Trae AI 团队

## 失败现象

租户设置页 `/tenant/{id}/settings/feature-params/` 的「API 端点(base_url)」保存后，输入框/落库值不再是用户填写的 URL。

现网例（company_id=`877397588196749312`）：

```text
https://api.deepseek.com/v1https://api.deepseek.com
```

## 失败环境
- 页面：公司智能体资源配置
- 组件：`FeatureParamsProvidersEditor` `base_url` input

## 排查过程

1. 读库 `cloud_tenant_feature_params.providers[].base_url` 为两段 `https://` 粘连。
2. 根因：选 deepseek 时前端预填 `https://api.deepseek.com/v1`，用户再粘贴自己的端点且未全选，两段叠在一起后保存。
3. 另有写路径 `normalizeProviderBaseURL` 把任意含 `/anthropic` 的 DeepSeek URL 改成官方 `/v1`，运营商自定义网关会被覆写。

## 解决方案

1. 不按运营商预填或改写 `base_url`（含 `/anthropic`）。
2. 仅当出现两个绝对 URL 粘连时保留最后一段。
3. GET/POST/env coerce / YAML 同步同一规则。

## 预防措施

1. 禁止按 provider 名改写用户端点。
2. 单测覆盖：原样保留 `/anthropic` 与自定义网关；粘连修复。

## 相关案例
- [99_deepseek_anthropic_base_url_404.md](./99_deepseek_anthropic_base_url_404.md)
