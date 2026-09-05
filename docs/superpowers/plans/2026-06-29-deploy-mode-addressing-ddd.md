# DDD 领域模型: DEPLOY_MODE 寻址模式切换

> 输入:
> - 设计文档: `.claude/skills/1-brainstorming-设计文档/design.md`
> - 价值流文档: `docs/superpowers/plans/2026-06-29-deploy-mode-addressing-value-stream.md`
> - NFR 澄清: `docs/superpowers/plans/2026-06-29-deploy-mode-addressing-nfr-clarification.md`
>
> 输出使用者: `/7-plans-实施计划`, `/8-build-构建`

## NFR 约束（取自 NFR 澄清文档）

| NFR 决策 | 模型影响 |
|----------|---------|
| 可用性 L1 — base.yaml 必需 | DeployMode / AddressingScheme 必须从 base.yaml 构造，无默认构造路径 |
| 可维护性 L2 — 向后兼容 | 模板变量解析为幂等纯函数 |
| 容错 L1 — 配置格式错误启动失败 | 构造时立即校验，不延迟到首次使用 |

## 限界上下文

```
部署配置上下文 (Deployment Configuration)
  职责: 管理服务寻址的单一真相源，将 DEPLOY_MODE 环境变量 + base.yaml
        解析为各服务可用的最终地址
  边界: 纯配置解析，不涉及运行时服务发现或健康检查
  通信: 通过 Python 模块导入被 conf_loader / conf-read / routes-to-apisix 消费
```

## 领域模型

### 值对象

#### DeployMode
```
DeployMode (Value Object)
  值: "domain" | "local"
  来源: DEPLOY_MODE 环境变量 || base.yaml mode 字段
  约束: 必须是 "domain" 或 "local" 之一
  行为: 无（不可变标签）
```

#### AddressingScheme
```
AddressingScheme (Value Object)
  属性:
    - mode: DeployMode
    - addresses: dict[str, str]  # {"api": "api.daydaymoney.com", "auth": "...", ...}
    - base_domain: str
  来源: base.yaml subdomains 段 (domain) 或 local 段 (local)
  约束: domain 模式下地址不含端口，local 模式下地址为 host:port
  行为:
    - resolve(key: str) -> str  # 按 key 返回地址
    - resolve_template(value: str) -> str  # 替换 ${subdomains.xxx}
```

### 领域服务

#### BaseYAMLLoader
```
BaseYAMLLoader (Domain Service)
  职责: 读取 conf/base.yaml，构造 DeployMode + AddressingScheme
  输入: base.yaml 文件路径
  输出: AddressingScheme
  规则:
    1. 解析 mode 字段 → DeployMode
    2. 根据 mode 展开对应段（subdomains / local）
    3. domain 模式: ${baseDomain} → BASE_DOMAIN 环境变量或 base.yaml baseDomain
    4. local 模式: 各 key 查找对应 *_ADDR 环境变量或 base.yaml local.* 默认值
  失败: YAML 解析失败 / 必需字段缺失 → 立即抛出，阻止启动
```

#### TemplateResolver
```
TemplateResolver (Domain Service)
  职责: 递归遍历配置 dict，替换所有 ${subdomains.xxx} 和 ${baseDomain} 占位符
  输入: config dict + AddressingScheme
  输出: 已解析的 config dict（新对象，不修改原对象）
  规则:
    - 只处理字符串值，非字符串原样保留
    - ${subdomains.xxx} → AddressingScheme.addresses[xxx]
    - ${baseDomain} → AddressingScheme.base_domain
    - 纯函数: 相同输入总是相同输出，无副作用
```

## 目录结构

```
runAll/scripts/domain/
├── __init__.py
├── value_objects/
│   ├── __init__.py
│   ├── deploy_mode.py          # DeployMode value object
│   └── addressing_scheme.py    # AddressingScheme value object
└── services/
    ├── __init__.py
    ├── base_yaml_loader.py     # BaseYAMLLoader domain service
    └── template_resolver.py    # TemplateResolver domain service
```

## 领域事件

无。配置解析是启动时的同步操作，不产生跨上下文事件。

## 自检

- [x] 所有文件位于 `domain/` 目录下
- [x] 领域层无 ORM 导入
- [x] 领域层无外部服务导入（yaml 解析通过接口注入）
- [x] 值对象不可变
- [x] 领域服务通过参数注入依赖，不隐式依赖全局状态
- [x] 纯函数（TemplateResolver）无副作用
- [x] 构造时校验（DeployMode / AddressingScheme 非法值立即拒绝）
