# 测试意图：打开容器页面前补齐安全组白名单

| 用例 | 层级 | Given | When | Then |
|------|------|-------|------|------|
| T1 normalize CIDR | Go unit | IP /32 /24 /0 | normalizePublicHostCIDR | 仅 host /32|/128 |
| T2 collect IPs | Go unit | body + XFF | collectEnsureClientIngressIPs | 去重合并 |
| T3 mock skip | Go unit | platform=mock | POST ensure-client-ingress | 200 skipped |
| T4 missing IP | Go unit | 无 body/XFF | POST | 400 |
| T5 open ensure then open | Vitest | mock apiFetch | openContainerPageWithIngressEnsure | POST ensure 后 openFn |
| T6 ensure fail still open | Vitest | apiFetch throw | open… | 仍 openFn |
