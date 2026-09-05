---
name: source-driven-development
description: 源码驱动开发 — 每个框架相关的代码决策必须由官方文档背书。使用官方文档验证 Go/Django/Vue/TypeScript 的 API 使用，防止基于过时记忆编码。适用于所有框架特定代码。
source: adapted from addyosmani/agent-skills
---

# 源码驱动开发（Source-Driven Development）

## 概述

**每个**框架相关的代码决策必须有官方文档支持。不从记忆中实现——验证、引用、让用户看到你的来源。训练数据会过时，API 会被废弃，最佳实践在演进。此技能确保代码可信，因为每个模式都追溯到可验证的权威来源。

## 流程

```
DETECT ──→ FETCH ──→ IMPLEMENT ──→ CITE
  │          │           │            │
  ▼          ▼           ▼            ▼
 检测      获取相关    遵循文档     展示来源
 技术栈    官方文档    模式实现
```

### Step 1: 检测技术栈版本

从依赖文件读取确切版本：

```
go.mod          → Go 版本 + 依赖
requirements.txt / pyproject.toml → Python/Django 版本
package.json    → Vue / Vite / Nuxt 版本
```

明确陈述发现：

```
STACK DETECTED:
- Go 1.24 (from go.mod)
- Django 5.0 (from pyproject.toml)
- Vue 3.5 (from package.json)
→ 拉取相关模式的官方文档。
```

### Step 2: 拉取官方文档

拉取你要实现的具体功能的文档页——不是首页，不是全站 — 是**相关的那一页**。

**来源层级（按权威度降序）：**

| 优先级 | 来源 | 示例 |
|--------|------|------|
| 1 | 官方文档 | go.dev/doc/, docs.djangoproject.com, vuejs.org |
| 2 | 官方 Blog / Changelog | go.dev/blog, blog.vuejs.org |
| 3 | Web 标准 | MDN, web.dev, html.spec.whatwg.org |
| 4 | 兼容性数据 | caniuse.com, node.green |

**非权威—绝不用作一手来源：**
- Stack Overflow 答案
- Blog 文章或教程（即使很流行）
- AI 生成的文档或摘要
- 你自己的训练数据（这是整个技能的重点——验证它）

### Step 3: 遵循文档模式实现

- 使用文档中的 API 签名，不是记忆中
- 如果文档展示了新做法，用新做法
- 如果文档废弃了某个模式，不要用废弃版本
- 如果文档没有覆盖某点，标记为未验证

**文档与既有代码冲突时：**

```
CONFLICT DETECTED:
既有代码使用 Django function-based views 处理表单，
但 Django 5.0 文档推荐使用 FormView class-based view。
(来源: docs.djangoproject.com/en/5.0/topics/class-based-views/)

选项:
A) 使用现代模式 (FormView) — 与最新文档一致
B) 匹配既有代码 — 与代码库一致
→ 将冲突暴露给用户，不擅自选择。
```

### Step 4: 引用来源

每个框架相关的模式都要有引用：

```go
// Go 1.24 的结构化日志使用 log/slog
// 来源: https://go.dev/blog/slog
logger.InfoContext(ctx, "task created",
    slog.String("taskId", task.ID),
    slog.String("userId", userID),
)
```

**引用规则：**
- 完整 URL，不缩短
- 优先使用深度链接 + 锚点
- 如果找不到某模式的文档，明确说明：

```
UNVERIFIED: 未找到此模式的官方文档。基于训练数据，
可能已过时。生产环境使用前请验证。
```

## 项目技术栈速查

| 技术 | 官方文档 | 版本检测 |
|------|---------|---------|
| Go | go.dev/doc/ | go.mod |
| Django | docs.djangoproject.com | pyproject.toml |
| Vue 3 | vuejs.org | package.json |
| TypeScript | typescriptlang.org/docs | package.json |
| PostgreSQL | postgresql.org/docs | Docker image tag |
| Redis | redis.io/docs | Docker image tag |
| Kafka | kafka.apache.org/documentation | Docker image tag |

## 红旗

- 写框架相关代码却不检查该版本的文档
- 用"我相信"、"我觉得"来描述 API 行为而非引用来源
- 不知道模式适用于哪个版本
- 引用 Stack Overflow 或 blog 代替官方文档
- 使用训练数据中的废弃 API
- 没有实际读取 go.mod/package.json 就写代码

## 验证

- [ ] 从依赖文件识别出框架和库版本
- [ ] 框架相关模式参考了官方文档
- [ ] 所有来源是官方文档，不是 blog 或训练数据
- [ ] 代码遵循当前版本文档的模式
- [ ] 非平凡决策包含来源引用（完整 URL）
- [ ] 没有使用废弃的 API
- [ ] 文档与代码冲突已暴露给用户
- [ ] 无法验证的内容明确标记为 UNVERIFIED
