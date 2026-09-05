# 测试意图：推荐资格管理分成比例固定 5%

## 测试目标

超管页面只展示固定 5%；不可编辑、不从配置读取。

## 测试分层

- 前端组件：`SystemAdminReferralManagement.ratios.test.js`
- 申请列表：`SystemAdminReferralApplicationsPanel.test.js`
- 后端：`taskBill/src/referral_config_test.go`、`taskReferral/src/handlers_test.go`、`taskReferral/src/referral_code_test.go`

## 用例矩阵

| ID | 场景 | 期望 |
|----|------|------|
| F1 | 打开比例卡 | 可见固定 `5%`；无输入、无保存、无 5%~30% 范围 |
| F2 | GET config 返回 12 | 页面仍展示 5%；不 POST `referral_rate_percent` |
| F3 | 无第二输入 | 不存在 `referral-profit-sharing-ratio-input` |
| F4 | 申请表 | 表头「分成比例」一列；行内展示接口回传值 |
| T54 | POST config 带 18 | 响应与后续读均为 5%；不 400 |
| T53 | 申请列表 | 每行展示固定 5%，即使库曾写入 12 |

## 通过标准

上述测例全绿；公网页比例卡只读 5%。
