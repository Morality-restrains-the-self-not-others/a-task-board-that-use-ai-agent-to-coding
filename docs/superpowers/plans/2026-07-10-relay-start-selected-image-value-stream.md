# 价值流：relay 启动所选镜像

## 主路径

```
用户选镜像 → PATCH 任务 container_image_id
    → 点「启动」
    → token-init + 凭证预检
    → POST relay-to-trae/start {installed_image_id, env}
    → Gateway 鉴权 + token-init + resolve-image
    → go_relayToTrae docker pull → docker run
    → status/SSE 回传 running
    → 用户点停止 → docker stop
```

## 测试点

| ID | 点 | 对应 |
|----|----|------|
| VS-1 | 启动 body 含 installed_image_id | 前端单测/意图 011 |
| VS-2 | resolve 失败拒绝 | Gateway 单测 |
| VS-3 | pull+run 所选 image | go_relayToTrae 单测 |
| VS-4 | stop 停容器 | go_relayToTrae 单测 |
