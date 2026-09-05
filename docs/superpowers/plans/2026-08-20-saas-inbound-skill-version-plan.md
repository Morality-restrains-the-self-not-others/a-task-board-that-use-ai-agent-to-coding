# 实施计划：容器→SaaS 接口版本

## 任务

- [x] D1 领域：`ValidateSaasInboundSkillVersion` + 事件常量 + 单测（红→绿）
- [x] D2 `versions.yaml` + skill md 标注 Version；Go 加载 Catalog
- [x] D3 GET `/api/ai-provider/saas-inbound-skill-versions/` + OpenAPI；md `?version=`
- [x] D4 `006_saas_inbound_skill_version.sql` ALTER + 回填 `1`
- [x] D5 store 读写新列；Create/Update 校验；发布事件
- [x] D6 厂商 POST/PUT 400 测例
- [x] F1 skill 页 badge + 版本选择器
- [x] F2 `SaasInboundSkillVersionSelect` 组件；VendorPortal 添加/编辑必选 + clickGuard
- [x] F3 Admin 审核表增加「接口版本」列；catalog JSON 字段
- [x] I1 更新 `frontend/src/skill.md`、intents 对照

命令（验收）：

```bash
cd taskAiProvider && go test ./domain ./src -count=1 -timeout 120s
cd taskAiProvider/frontend && node --test tests/saasMachineContainerSkill.unit.test.js tests/saasInboundSkillVersion.unit.test.js
python3 -c "import yaml; yaml.safe_load(open('docs/skills/saas-container/versions.yaml'))"
```
