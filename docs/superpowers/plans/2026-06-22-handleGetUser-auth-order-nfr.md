# NFR: Fix handleGetUser Auth Order

> Security L3 (auth domain). Single-function fix. No domain model impact.

## NFR 概览

| 类别 | 等级 | 量化 |
|------|------|------|
| 安全性 | L3 | Internal secret 必须在无 bearer token 时也能通过认证 |

## 跳过声明

性能、可伸缩性、可用性、可观测性、合规、可维护性、数据一致性、容错 — 均跳过。纯代码逻辑修复，无架构/性能/数据变更。
