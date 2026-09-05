# 镜像组图标 — 领域模型

- **日期**: 2026-08-28
- **限界上下文**: AI Provider Marketplace（taskAiProvider）

## 聚合

**ImageGroup**（已有）：`ID`, `VendorID`, `Name`, `Description`, **`IconFileKey`**。

创建/更新不变量：Name、Description、IconFileKey 均非空白；IconFileKey 必须归属该 Vendor（`OwnsFileKey`）。

## 值对象

**ImageGroupIcon**：扩展名 ∈ {`.png`,`.jpg`,`.jpeg`,`.webp`}，size ∈ (0, 512KiB]，content-type 与扩展名一致。

## 端口

复用 `domain.VendorDocStore`（SaveLocal / PresignPut / Head / Open）。kind = `image_group_icon`。

## 领域事件

| 事件 | 何时 | payload（无文件字节） |
|------|------|----------------------|
| `ImageGroupIconUploaded` | upload-complete | vendor_id, file_key（可截断日志） |
| `ImageGroupCreated` | POST 组成功 | group_id, vendor_id |
| `ImageGroupUpdated` | PUT 组成功 | group_id, vendor_id |

投递：既有 `EventBus`（LogEventBus；无 Kafka 配置）。无独立消费者。

## 查询

`PublicImageGroupIcon(groupID)`：按组取 key → Open(vendorID, key) → 字节流。
