# 设计文档：密码重置邮件链接使用可配置的外部域名

**日期:** 2026-06-22
**状态:** 已批准

---

## 1. 问题陈述

### 现象

用户在 `http://183.250.1.132:4000/auth/login/` 点击密码重置，收到的邮件中链接为：
```
http://127.0.0.1:4000/auth/reset-password/<token>/
```

`127.0.0.1` 是回环地址——用户从邮件客户端无法访问。

### 根因

密码重置 URL 通过 `settings_manager.get_frontend_domain()` 构造：

```python
# core/config/settings_manager.py:211-216
def get_frontend_domain(self) -> str:
    frontend_config = self.get_frontend_config()
    host = frontend_config.get("host", "localhost")   # ← 127.0.0.1 (内部绑定地址)
    port = frontend_config.get("port", 3000)
    return f"http://{host}:{port}"                    # ← http://127.0.0.1:4000
```

数据来源于 `conf/frontend/vue/config.yaml`:
```yaml
host: 127.0.0.1          # 内部监听地址
port: 4000
apiBaseUrl: http://183.250.1.132:18081   # 前端调用的外部 API 网关
allowedHost: http://daydaymoney.com           # 生产环境前端域名
```

**此配置中没有「前端自身外部 URL」的概念。** `host` 是进程绑定地址，不应用于面向用户的邮件。

### 调用链

```
前端: POST /api/accounts/users/send_password_reset_link/
  → taskAuth (Go) 处理
    → 回调 Django: POST /api/internal/taskauth/post-password-reset-link/
      → settings_manager.get_frontend_domain() → "http://127.0.0.1:4000"
      → Kafka EMAIL_SENT
```

---

## 2. 解决方案

### 核心策略：在 vue 配置中增加 `publicBaseUrl` 字段

项目中已有先例——其他服务使用 `publicBaseUrl` / `publicBase` 表示外部可访问 URL：

| 服务 | 字段 | 值 |
|------|------|-----|
| `task-ai-endpoint` | `publicBaseUrl` | `http://127.0.0.1:8013` |
| `task-gateway` | `publicBase` | `http://183.250.1.132:18081` |

### 配置

```yaml
# conf/frontend/vue/config.yaml
publicBaseUrl: http://183.250.1.132:4000    # 新增：前端外部 URL
```

### 代码

```python
# settings_manager.py get_frontend_domain()
def get_frontend_domain(self) -> str:
    frontend_config = self.get_frontend_config()
    public_url = frontend_config.get("publicBaseUrl", "")
    if public_url:
        return public_url.rstrip("/")
    host = frontend_config.get("host", "localhost")
    port = frontend_config.get("port", 3000)
    return f"http://{host}:{port}"
```

### 回退

若 `publicBaseUrl` 缺失 → 回退至现有行为 `http://{host}:{port}`。

---

## 3. 文件变更

| 文件 | 变更 |
|------|------|
| `conf/frontend/vue/config.yaml` | 新增 `publicBaseUrl: http://183.250.1.132:4000` |
| `core/config/settings_manager.py` | `get_frontend_domain()` — 优先 `publicBaseUrl` |

---

## 4. 领域概念

无需记录——纯配置修复。

## 5. 价值流影响

跳过——配置缺陷修复，无新业务概念。
