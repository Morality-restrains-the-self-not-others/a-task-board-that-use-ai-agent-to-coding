# 测试意图：镜像容器技能列表

## 覆盖

| ID | 场景 | 测例落点 |
|----|------|---------|
| T1 | YAML 第一项为 default，非法 name 失败 | `parse_image_skills_test.go` |
| T2 | OCI tar 命中 `/app/imageSkills.yaml` | `extract_image_skills_test.go` |
| T3 | 厂商 POST 镜像触发抽取并落 JSON | `vendor_container_image_skills_test.go` |
| T4 | 公开 catalog 含 image_skills | `store_public_catalog_test.go` |
| T5 | 安装快照拷贝技能；空则再抽 | `installed_image_skills_extract_test.go` |
| T6 | 市场组件渲染技能列表 | `ImageSkillsList.test.js` |
| T7 | 任务描述 `/skill` 高亮仅匹配所选镜像 | `imageSkills.test.js` |
| T10 | 已绑定镜像时空描述点击技能 chip 写入 `$镜像 /技能` 且不解绑 | `TaskDescriptionSkillField.test.js` / `CreateTaskMentionSync.test.js` |
| T8 | 评论 @ 后 `/` 补全并写入 mentions.skill | `commentImageMentionComposer.test.js` |
| T9 | mentions.skill 缺省填 default；未知 skill 400 | `image_mention_test.go` |
