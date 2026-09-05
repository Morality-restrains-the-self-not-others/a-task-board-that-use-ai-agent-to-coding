---
name: security-and-hardening
description: 安全加固规范。适用于处理用户输入、认证授权、数据存储、外部集成、文件上传、支付/PII 数据等场景。覆盖 OWASP Top 10、STRIDE 威胁建模、供应链安全、SSRF 防护。
source: adapted from addyosmani/agent-skills
---

# 安全加固（Security & Hardening）

## 概述

将每个外部输入视为敌意，将每个密钥视为神圣，将每个授权检查视为强制。安全不是一个阶段——它是每一行涉及用户数据、认证、外部系统代码的约束。

## 流程：威胁建模优先

不基于威胁模型的控制措施只是猜测。加固前花五分钟以攻击者视角思考：

1. **标出信任边界** — 不受信任的数据在哪里进入系统？HTTP 请求、表单字段、文件上传、webhook、第三方 API、消息队列，以及 **LLM 输出**。每个边界都是攻击面
2. **命名资产** — 什么值得窃取或破坏？凭证、PII、支付数据、管理员操作、资金流动
3. **对每个边界运行 STRIDE** — 快速视角，不是仪式：

| 威胁 | 问题 | 典型缓解 |
|------|------|---------|
| **S**poofing | 能否模拟用户/服务？ | 认证、签名验证 |
| **T**ampering | 数据在传输/存储中被篡改？ | 完整性检查、参数化查询、HTTPS |
| **R**epudiation | 操作可被否认？ | 安全事件审计日志 |
| **I**nfo disclosure | 数据泄漏？ | 加密、字段白名单、通用错误 |
| **D**enial of Service | 可被压垮？ | 限流、输入大小上限、超时 |
| **E**levation | 用户获得不应有的权限？ | 授权检查、最小权限 |

4. **为每个功能写 abuse case（滥用用例）** — "如果我要滥用这个功能该怎么做？" — 然后把它作为第一个测试

## 三层边界系统

### Always Do（无例外）

- **验证所有外部输入**在系统边界（API handler、表单 handler）
- **参数化所有数据库查询** — 绝不将用户输入拼接到 SQL
- **编码输出**防 XSS（使用框架自动转义，不绕过）
- **使用 HTTPS** 进行所有外部通信
- **哈希密码**使用 bcrypt/scrypt/argon2（绝不存储明文）
- **设置安全头**（CSP, HSTS, X-Frame-Options, X-Content-Type-Options）
- **使用 httpOnly, secure, sameSite cookies** 存储 session
- **每次发版前运行依赖审计**（Go: `govulncheck`; Python: `pip-audit`; Node: `npm audit`）

### Ask First（需人工审批）

- 新增认证流程或修改认证逻辑
- 存储新类别的敏感数据（PII、支付信息）
- 新增外部服务集成
- 修改 CORS 配置
- 新增文件上传处理
- 修改限流或节流
- 赋予提升权限或角色

### Never Do

- **绝不在源码中硬编码密钥**（API keys、密码、token、私钥）。须写入 gitignored `conf-local/`（与 `conf/` 同相对路径）、环境变量或 KMS（元规则第 57/58 条 / ADR-0046 / ADR-0054）。**绝不提交** `.env` / 已跟踪 YAML 中的机密；加载器不读 `config.local.yaml`
- **绝不记录敏感数据**（密码、token、完整信用卡号）
- **绝不信任客户端验证**作为安全边界
- **绝不为了方便禁用安全头**
- **绝不使用 `eval()` 或 `innerHTML`** 处理用户数据
- **绝不在客户端可访问存储中保存 session**（localStorage 存 auth token）
- **绝不向用户暴露堆栈追踪**或内部错误详情

## Go 安全实践

### SQL 注入防护

```go
// BAD: 字符串拼接
query := fmt.Sprintf("SELECT * FROM users WHERE id = '%s'", userID)

// GOOD: 参数化查询
row := db.QueryRowContext(ctx,
    "SELECT * FROM users WHERE id = $1", userID)
```

### 输入验证

```go
// Go: 在 handler 边界验证
type CreateTaskInput struct {
    Title       string `json:"title" validate:"required,min=1,max=200"`
    Description string `json:"description" validate:"max=2000"`
    Priority    string `json:"priority" validate:"oneof=low medium high"`
}

func (h *Handler) CreateTask(w http.ResponseWriter, r *http.Request) {
    var input CreateTaskInput
    if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
        http.Error(w, `{"error":{"code":"INVALID_JSON"}}`, http.StatusBadRequest)
        return
    }
    if err := h.validate.Struct(input); err != nil {
        http.Error(w, `{"error":{"code":"VALIDATION_ERROR"}}`, http.StatusUnprocessableEntity)
        return
    }
    // input 已通过类型和验证
}
```

### SSRF 防护

服务端拉取用户影响的 URL（webhook、"从 URL 导入"、图片代理、链接预览）时，攻击者可以瞄准内部服务：

```go
// BAD: 直接拉取用户提供的任何东西
resp, _ := http.Get(req.FormValue("url"))

// GOOD: 白名单 scheme + host，拒绝私有 IP
var allowedHosts = map[string]bool{"hooks.example.com": true}

func fetchUserURL(rawURL string) error {
    u, err := url.Parse(rawURL)
    if err != nil || u.Scheme != "https" || !allowedHosts[u.Hostname()] {
        return errors.New("url not allowed")
    }
    // 解析 DNS 并检查所有解析结果是否为公网 IP
    ips, _ := net.LookupIP(u.Hostname())
    for _, ip := range ips {
        if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
            return errors.New("private/reserved IP rejected")
        }
    }
    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse // 禁止跟随重定向
        },
    }
    resp, err := client.Get(u.String())
    // ...
}
```

## Django 安全实践

### 安全配置

```python
# settings.py
SESSION_COOKIE_HTTPONLY = True
SESSION_COOKIE_SECURE = True
SESSION_COOKIE_SAMESITE = 'Lax'
CSRF_COOKIE_HTTPONLY = True
CSRF_COOKIE_SECURE = True
SECURE_HSTS_SECONDS = 31536000
SECURE_HSTS_INCLUDE_SUBDOMAINS = True
SECURE_CONTENT_TYPE_NOSNIFF = True
SECURE_BROWSER_XSS_FILTER = True
X_FRAME_OPTIONS = 'DENY'
```

### 访问控制

```python
# views.py
@login_required
def update_task(request, task_id):
    task = get_object_or_404(Task, id=task_id)
    # 检查用户拥有此资源——不只是已认证
    if task.owner_id != request.user.id:
        return JsonResponse({
            'error': {'code': 'FORBIDDEN', 'message': 'Not authorized'}
        }, status=403)
    # ...
```

## AI/LLM 功能安全

如果应用调用 LLM（chatbot、摘要器、agent、RAG），继承新的攻击面。参考 [OWASP Top 10 for LLM Applications](https://genai.owasp.org/llm-top-10/)：

- **模型输出视为不受信任的输入（LLM05）** — 绝不将 LLM 输出直接传入 SQL、shell、`innerHTML`
- **假设 prompt 可被劫持（LLM01）** — system prompt 不是安全边界；用代码执行权限检查
- **不将密钥和跨租户数据放入 prompt（LLM02/LLM07）**
- **约束工具/agent 权限（LLM06）** — 作用域最小化，破坏性操作需确认
- **限制消费（LLM10）** — 上限 token、请求频率、循环/递归深度

## 依赖审计决策树

```
审计工具报告一个漏洞
├── 严重: critical 或 high
│   ├── 脆弱代码在生产/构建环境中可达？
│   │   ├── YES → 立即修复（更新/补丁/替换依赖）
│   │   └── NO → 尽快修复但不阻塞
│   ├── 有修复版本？
│   │   ├── YES → 更新到已修复版本
│   │   └── NO → 寻找 workaround 或考虑替换
├── 严重: moderate → 下一个发版周期修复
└── 严重: low → 跟踪，定期更新时修复
```

## 审查清单

```
### 认证
- [ ] 密码用 bcrypt/scrypt/argon2 哈希（≥12 rounds）
- [ ] Session token 为 httpOnly, secure, sameSite
- [ ] 登录有速率限制

### 授权
- [ ] 每个端点检查用户权限
- [ ] 用户只能访问自己的资源
- [ ] 管理员操作需验证管理员角色

### 输入
- [ ] 所有用户输入在边界验证
- [ ] SQL 查询参数化
- [ ] HTML 输出编码/转义
- [ ] 服务端 URL 拉取白名单检查（无 SSRF）

### 数据
- [ ] 代码/版本控制中无密钥
- [ ] API 响应排除敏感字段
- [ ] PII 静态加密

### 基础设施
- [ ] 安全头已配置（`curl -I` 检查）
- [ ] CORS 限制到已知 origin
- [ ] 依赖无 reachable critical/high 漏洞
- [ ] 错误消息不暴露内部细节
```
