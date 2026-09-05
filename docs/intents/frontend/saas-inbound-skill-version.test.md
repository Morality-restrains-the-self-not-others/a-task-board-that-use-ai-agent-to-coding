# 测试意图：容器→SaaS 接口版本（厂商门户）

## 对应功能意图

`saas-inbound-skill-version.intent.md`

## 用例

| ID | 场景 | 期望 |
|----|------|------|
| FE-1 | skill 页加载成功 | 出现版本 badge，正文来自 markdown |
| FE-2 | 切换已发布 version query | 重新拉取 `/saas-machine-container.md?version=` |
| FE-3 | 添加镜像未选接口版本 | 前端拦截，不发 POST |
| FE-4 | 添加镜像已选 v1 | POST body 含 `saas_inbound_skill_version: "1"` |
| FE-5 | 保存按钮连点 | 第二次被 clickGuard 跳过 |
| FE-6 | catalog API 200 | 下拉可选已发布版本；不再出现 `skill version catalog unavailable` |

## 自动化落点

- `taskAiProvider/frontend/tests/saasMachineContainerSkill.unit.test.js`
- `taskAiProvider/frontend/tests/saasInboundSkillVersion.unit.test.js`
