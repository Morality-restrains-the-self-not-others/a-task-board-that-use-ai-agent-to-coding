# 注册邀请码 — 价值流

## 主价值流：受邀注册

1. Admin 开启并设 daily_quota  
2. 老用户 Profile 申请码（消耗当日配额）  
3. 新用户 Login/Register 填写邀请码  
4. taskAuth 校验并核销 → 创建用户 → 事件 REDEEMED  
5. Admin 关系表可见 issuer→code→redeemer  

## 测试点（映射 test-intent）

- VS-T1 关闭不挡注册  
- VS-T2 开启无码挡注册  
- VS-T3 配额耗尽挡申请  
- VS-T4 一码一用  
- VS-T5 Admin 关系可读  

## CRG 社区对齐

图未覆盖 auth 社区；价值流域对齐 taskAuth 注册流。
