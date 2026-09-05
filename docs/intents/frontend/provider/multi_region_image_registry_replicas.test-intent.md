# 测试意图：厂商按区域登记容器仓库公网地址

## 测试目标

验证区域运行环境表单携带 `registry_public_url`、展示推导内网、VPC 地址错误富化。

## 测试分层

- 单元：`taskAiProvider/frontend/tests/` 区域环境 payload / 只读 intranet
- 与 `privateRegistryHint.unit.test.js` 协同：VPC 提交被拒

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 表单填公网 ACR | build association payload | 含 `registry_public_url`，不含手写 intranet |
| T2 | API 返回 intranet_url | 渲染 | 只读展示该值 |
| T3 | 空仓库字段 | payload | 无 replica 键或空字符串，CSI 仍可保存 |
| T4 | VPC URL 400 | 错误节点 | 文案无法触及 + data-traceId |

## 数据与环境

不打真实仓库；mock association API。

## 通过标准

上表单测全绿。
