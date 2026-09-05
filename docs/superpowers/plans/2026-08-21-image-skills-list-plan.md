# 实施计划：镜像容器技能列表

## Task 1: 解析 VO + 单测
- 文件：`taskAiProvider/infrastructure/parse_image_skills.go` + `_test.go`；domain 常量路径
- 验证：`go test ./infrastructure -run ParseImageSkills`

## Task 2: OCI 抽取
- 复用 layer walk，路径 `/app/imageSkills.yaml`
- 验证：`go test ./infrastructure -run ExtractImageSkills`

## Task 3: 厂商保存绑定 + 事件
- schedule 与 autoRun 并行；列 migration 007
- 验证：`go test ./src -run SkillsExtract`

## Task 4: catalog + Cloud 快照
- 公开 JSON；Cloud 035 列；安装拷贝/再抽
- 验证：catalog test + installed extract test

## Task 5: mentions.skill
- ImageMention.Skill；校验属于列表或格式
- 验证：`go test ./domain -run Skill`

## Task 6: FE 市场/创建任务/评论
- ImageSkillsList；描述高亮；slash picker
- 验证：对应 vitest

## Task 7: 文档 + demo yaml + IMAGE_SKILL
- saas-machine-container.md；trae-agent COPY
- 验证：markdown 含路径约定

## Task 8: 价值流图测试点
- `docs/flows/value-stream-test-integration.wsd`
