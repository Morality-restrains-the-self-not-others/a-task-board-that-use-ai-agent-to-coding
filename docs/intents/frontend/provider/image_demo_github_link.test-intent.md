# 测试意图：厂商门户顶栏镜像 Demo GitHub 链接

## 测试目标

验证镜像市场顶栏「镜像Demo」为真实外链，指向 `https://github.com/task2money/trae-agent`。

## 测试分层

- 单元（前端）：`taskAiProvider/frontend/tests/imageDemoLink.unit.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | 常量模块 | 读取 href/文案 | href 为 `https://github.com/task2money/trae-agent`；文案为「镜像Demo」 |
| T2 | `App.vue` | 读取模板 | 存在 `data-testid="nav-image-demo"`、绑定 `IMAGE_DEMO_HREF`、`target="_blank"`、`rel` 含 `noopener noreferrer`，无 `@click.prevent` |

## 数据与环境

- 前端单测读 `App.vue` 源码与常量模块，无需浏览器。

## 通过标准

```
cd taskAiProvider/frontend && npm run test:unit -- tests/imageDemoLink.unit.test.js
```

全绿。
