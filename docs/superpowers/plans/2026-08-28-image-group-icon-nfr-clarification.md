# 镜像组图标 — NFR 澄清

- **日期**: 2026-08-28
- **价值流**: `docs/superpowers/plans/2026-08-28-image-group-icon-value-stream.md`

## 路径分片键强制审视

| 路径 | 分片 ID | 判定 | 动作 |
|------|---------|------|------|
| POST `/api/vendor/image-groups/` | 无 tenant；身份=vendor JWT | L0 | 镜像组按厂商隔离、年增量远低于分片阈值；升级触发：单表 >100 万组 |
| POST `.../icon-upload-url/` 等同路径 | 无 | L0 | 同上；对象键含 `{userId}` |
| GET `/api/public/image-groups/{id}/icon` | 路径含 group id，非租户键 | L0 | 公开只读、按组主键点查；升级触发：CDN 分流 |
| GET `/api/public/catalog/` | 既有 | L0 | 本增量只加字段 |
| 前端 `/` 厂商门户、主站 ImageMarket | 无新路由 | L0 | — |

可伸缩性相关类别：书面 L0（小表 + 点查）。

## 幂等性强制审视

| 路径 | 副作用 | 重复触发 | 业务边界 | 幂等键 | 重放语义 |
|------|--------|----------|----------|--------|----------|
| GET public icon / catalog | 无 | — | — | L0 只读 | — |
| POST icon-upload-url | 签发 key（COS 预签名） | 双击 | 每次签发新 object key | Idempotency-Key 防双击；对象允许新 key | 返回新 URL |
| PUT local-put / COS PUT | 写对象 | 重试同 file_key | file_key | 覆盖写同 key | 成功空操作等价 |
| POST upload-complete | Head + 事件 | 重试 | file_key | Head 成功即 200 | 重复 complete 200 |
| POST/PUT image-groups | 写组行 | 双击保存 | 创建=新组；更新=group id + 字段 | 前端 createClickGuard + Idempotency-Key；创建无唯一名约束（既有） | 更新覆盖；创建可能两条（既有行为，不在本增量改名唯一） |

资金/云资源：否。一致性 L2（标准）。

## 其它 NFR

| 类别 | 等级 | 说明 |
|------|------|------|
| 安全 | L2 | 类型/大小校验；无 SVG；公开流不接受客户端 key；日志不打完整对象内容 |
| 性能 | L1 | 图标 ≤512KB；Cache-Control + `?h=` |
| 可用性 | L1 | 图标 404 时 UI 占位，不阻断组列表 |
| 可观测性 | L2 | event=ImageGroupIconUploaded / ImageGroupCreated / ImageGroupUpdated + trace |

## 领域模型影响

ImageGroup 聚合增加必填值对象 IconFileKey；公开读为查询侧。
