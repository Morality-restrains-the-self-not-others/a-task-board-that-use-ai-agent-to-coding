# 测试意图：赠送资源/订单记录下拉展示租户邮箱与手机号

## 测试目标

下拉与已选摘要展示真实邮箱/手机；空值不渲染。

## 测试分层

| 层 | 位置 |
|----|------|
| 工具单测 | `taskFE/app/src/utils/tenantOptionContact.test.js` |
| 组件单测 | `taskFE/app/src/views/SystemAdminGrantPoints.tenant-options.unit.test.js` |

## 用例矩阵

| 场景 | 期望 |
|------|------|
| phone+email 均有 | 下拉含两者，以间隔符连接 |
| 仅 email | 只显示 email |
| 值为「无」或空 | 不显示联系行 |
| 选中后 | 「已选」含联系方式 |

## 数据与环境

jsdom + vitest；mock `apiFetch` 返回带 phone/email 的 tenant-options。

## 通过标准

上述测例全绿。
