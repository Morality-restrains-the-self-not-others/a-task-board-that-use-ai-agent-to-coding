# 测试意图：厂商证照 COS 预签名直传

## 测试目标

覆盖预签名签发、浏览器 PUT 契约、Head 确认、路径规则写回、本地 file_key 回退，以及禁止密钥/签名 URL 入日志。

## 测试分层

- 单元：pathRule 渲染与拒绝；VendorDocStore 假 COS；handler 契约
- 组件：主站镜像市场 `ImageMarketVendorApply` 走 upload-url → 直传 → complete
- 不跑真实证件、不默认打真实桶（假客户端）

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| T1 | upload-url 合法 png | 200 + file_key + PUT url + 加密头 |
| T2 | upload-url 超限/非法类型 | 400 |
| T3 | upload-complete Head 命中且属当前用户 | 200；投递 VendorDocumentUploaded |
| T4 | upload-complete 对象不存在 / 越权 key | 400 |
| T5 | backend=cos 打旧 multipart | 410 |
| T6 | 申请提交 Head 通过 | 200 pending |
| T7 | 存量本地 file_key staff 下载 | 200（COS miss 回退本地） |
| T8 | 非 staff 下载 | 401/403 |
| T9 | FE：upload-url 后 PUT 再 complete | 断言三次调用顺序 |
| T10 | FE：齐全后 body 含 COS file_key | postBody 断言 |
| T11 | pathRule 含 `..` 或未知占位符 | 400，文件未改 |
| T12 | PATCH 合法 pathRule | 写 vendor-docs-path.yaml；热加载；VendorDocPathRuleUpdated |

## 数据与环境

虚构字节；COS 用 fake；`config.local.yaml` 不进仓库。

## 通过标准

上表全绿；日志 fixture 不含 Secret 与预签名 query。
