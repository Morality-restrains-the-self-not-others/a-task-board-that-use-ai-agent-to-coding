# NFR 澄清 — 人员页成员/邀请/分组迁 Go

日期：2026-07-19  
默认档位：L2 Standard；鉴权相关 L3

| 类别 | 级别 | 要求 | 喂给 DDD/实现 |
|------|------|------|----------------|
| 安全/鉴权 | L3 | Gateway JWT；admin 门禁；internal secret；禁止伪造 X-User-Id | Policy 服务 + middleware |
| 正确性 | L2 | 契约兼容前端；创建者不可删/禁/改角色 | 不变式在领域服务 |
| 性能 | L2 | 列表 P95 &lt; 300ms（本地 sqlite） | 索引 company_id/user_id |
| 可用性 | L2 | 独立进程；Django 宕不阻断只读公司校验失败需明确错误 | 超时/错误码 |
| 可观测 | L2 | tracelog + data-traceId；level 小写 | shareLib/tracelog |
| 迁移安全 | L2 | 切流窗口无双写；import 幂等（按 id） | import API |
| 兼容 | L2 | URL/JSON 字段与 Django 现网一致 | 序列化对齐 |

## 明确不做（本迭代）

- 多区域复制、强一致跨库事务
- 成员头像上传迁对象存储（保留字段/空 URL）
