# 价值流 — 自动 SG 入网白名单

## 主价值流：自动创建安全组并启动

```
用户打开 task-detail → 硬件配置选「自动创建安全组」→ 启动
  → taskCloudService 解析 client_public_ip
  → 发布 CLOUD_SERVER_START_AUTO
  → taskEvents 创建/复用 SG（仅用户 IP/32；撤销 0.0.0.0/0）
  → CLOUD_SERVER_STARTED → RunInstances
  → DescribeInstances 得公网 IP → 追加服务器 IP/32
  → 用户仅从本人公网 IP 访问容器端口
```

## 测试点

| ID | 测试点 | 对应用例 |
|----|--------|----------|
| VS-SG-1 | 自动 SG 规则不含 0.0.0.0/0 | unit: defaultAutoSGIngressRules |
| VS-SG-2 | 含用户 IP/32 | unit + startauto handler |
| VS-SG-3 | 启动后含服务器 IP/32 | unit: TightenAutoSG |
| VS-SG-4 | 复用旧全开 SG 时撤销全开 | unit: ensureWhitelistIngress |
| VS-SG-5 | 非 auto SG 路径行为不变 | existing start-vm tests |
| VS-SG-6 | 前端提示不再承诺「开放所有端口」 | UI 文案 |
