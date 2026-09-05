# 计划：relay token-init / env-prepare / precheck 迁 TCG

**日期：** 2026-07-09  
**设计：** `docs/superpowers/specs/2026-07-09-relay-token-init-tcg-routing-design.md`

## Tasks

- [x] 扩展 `taskGateway/routes/routes.yaml` 三 URI 族并再生 apisix.yaml
- [x] TCG：`credentialRepoCloneCredentials` + `handleRelayTokenInit/EnvPrepare/Precheck`
- [x] Django `@require_django_forward` 三 action → 410
- [x] 单测：lifecycle action / path parse / handler mock chain
- [ ] Playwright：点启动 → token-init≠501 → 克隆完成信号
- [ ] 热更新 TCG + APISIX 并冒烟验证

## 验证命令

```bash
cd taskContainerGateway && go test ./src/ -count=1 -run 'Relay|TokenInit|EnvPrepare|Precheck'
cd taskGateway && bash run.sh routes-apply
# 冒烟
curl -sS -X POST '.../relay-to-trae/token-init/' -H "Authorization: Token $TOK" ...
```
