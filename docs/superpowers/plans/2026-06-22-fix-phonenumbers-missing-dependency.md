# Fix: `phonenumbers` 依赖缺失导致 saas-backend 启动失败

**日期:** 2026-06-22
**类型:** Bug Fix (依赖配置遗漏)
**影响页面:** `http://183.250.1.132:9999/`

---

## 问题诊断

### 报错信息

```
File "/tmp/ram-work/task2app/Saas_project/accounts/phone_normalize.py", line 5, in <module>
    import phonenumbers
ModuleNotFoundError: No module named 'phonenumbers'
```

### 调用链

```
manage.py → django.setup() → apps.populate()
  → accounts/models/__init__.py
    → accounts/models/login_method.py
      → accounts/phone_normalize.py
        → import phonenumbers  ← 报错点
```

### 根因分析

| 文件 | `phonenumbers` 状态 |
|------|---------------------|
| `requirements/base.txt` | ✅ 已声明: `phonenumbers>=8.13.0,<9` |
| `requirements/test.txt` | ✅ 通过 `-r base.txt` 间接包含 |
| `requirements/run.txt` | ❌ **缺失** — 独立扁平列表，未包含 `phonenumbers` |
| `requirements/unit.txt` | ❌ 缺失 |
| `requirements/auto.txt` | ❌ 缺失 |
| `.venvs/run` (生产环境) | ❌ 未安装 |
| `.venvs/unit` (单元测试) | ✅ 已安装 `8.13.55` |

**根因**: `phonenumbers` 在 `base.txt` 中声明为共享依赖，但 `run.txt` 作为独立冻结列表未将其包含，导致运行虚拟环境 `.venvs/run` 中未安装该包。

### `phone_normalize.py` 中的使用

该模块是实现手机号规范化（短信验证码/登录/国际号码解析）的核心工具，涉及：
- `canonical_phone_for_sms_and_login()` — 中国大陆11位 / 国际 E.164 规范化
- `split_country_calling_code_and_national()` — 使用 `phonenumbers.parse()` 拆解国号与国内号码
- `format_phone_e164_cn()` — 短信网关手机号格式化

`phonenumbers` 被用于 `split_country_calling_code_and_national()` 中解析 E.164 号码（`phonenumbers.parse()`），是整个 `accounts` 模块启动时必需的依赖。

---

## 修复方案

### 方案: 在 `run.txt` 中补充 `phonenumbers` 并安装

**操作步骤：**

1. **编辑 `requirements/run.txt`** — 添加 `phonenumbers==8.13.55`（与 `.venvs/unit` 中已安装版本一致，满足 `base.txt` 约束 `>=8.13.0,<9`）
2. **同步安装** — 在 `.venvs/run` 虚拟环境中 `pip install phonenumbers==8.13.55`
3. **验证** — 重启 saas-backend 确认启动成功

**影响范围：**
- 文件变更: `task2app/Saas_project/requirements/run.txt` — 新增 1 行
- 环境变更: `.venvs/run` 新增 `phonenumbers` 包
- 无代码逻辑变更，无接口变更，无测试影响

**风险:** 极低。仅补充遗漏的依赖声明，不改变业务逻辑。

---

## 域概念清单

本次为依赖修复，不涉及业务域变更：

- **Bounded Contexts:** 无变更 — 属于 accounts 上下文的已有基础设施
- **Key Entities:** 无变更 — `LoginMethod` 实体已有 `phone_normalize` 依赖
- **Domain Events:** 无变更

---

## 价值流影响

无现有价值流受影响。此为基础设施缺陷修复（依赖声明遗漏），不改变任何价值流步骤、字段或测试。

---

## 总结清单

- 修复方案: 唯一的方案 — 在 `run.txt` 中补上 `phonenumbers` 依赖并安装到 `.venvs/run`
