# 测试意图：厂商门户可查看容器 → SaaS 接口 Skill

## 测试目标

验证公开 `GET /saas-machine-container.md` 返回仓库 SSOT 正文，且厂商门户顶栏有真实 href 指向该路径。

## 测试分层

- 单元（Go）：`taskAiProvider/src/saas_machine_container_skill_test.go`
- 单元（前端）：`taskAiProvider/frontend/tests/saasMachineContainerSkill.unit.test.js`

## 用例矩阵

| ID | 给定 | 当 | 则 |
|----|------|----|----|
| T1 | monorepo 含 skill SSOT | `GET /saas-machine-container.md` | 200；`Content-Type` 含 `text/plain`；正文含 `SaaS Machine Container Skill` 与 `server-container-token` |
| T2 | 同上 | `POST /saas-machine-container.md` | 405 |
| T3 | `App.vue` | 读取模板 | 存在 `<a href="/saas-machine-container.md"`，无 `@click.prevent` |
| T4 | 常量模块 | 读取 href | 值为 `/saas-machine-container.md` |

## 数据与环境

- Go 测试走 `FindMonorepoRoot` 读真实 SSOT，不造假文件内容。
- 前端单测读 `App.vue` 源码与常量模块，无需浏览器。

## 通过标准

```
cd taskAiProvider/src && go test -count=1 -run TestHandleSaasMachineContainerSkill .
cd taskAiProvider/frontend && npm run test:unit -- saasMachineContainerSkill.unit.test.js
```

全绿。
