# [运行时] 功能参数支持模型列表：中文逗号未分割成多个模型

## 基本信息

- 版本：1.0.0
- 创建日期：2026-07-20
- 最后修改：2026-07-20
- 维护者：Trae AI 团队

## 现象

- 页面：`/tenant/{id}/settings/feature-params/`（公司环境变量 / LLM 供应商配置）
- 「支持模型列表」textarea 输入 `gpt-4.1，gpt-4.1-mini`（中文逗号）后保存
- 候选模型下拉 / 校验把整段当作**一个**模型名，而非两个

## 根因

前端多处 `parseSupportedModels`（及保存前 `toProviderPayload`）使用正则 `/[\n, ]/`，仅识别换行、英文逗号与空格，**未包含中文逗号 `，`**（及常见分号）。

后端（Django `normalize_provider_entry`、Go `normalizeProviderEntry`）在收到字符串形态 `supported_models` 时同样只按 `\n`→`,` 再 `split(",")`，存在相同盲区。

## 解决方案

1. 前端将解析收敛到 `featureParamsModelValidation.js` 的 `parseSupportedModels`，分隔符：`/[\n\r,，;； \t]+/`。
2. 公司/工作空间/个人设置页与任务详情模型选项统一 import 该函数。
3. Python `parse_supported_models`、Go `parseSupportedModelsField` 与前端对齐（防御：API 直传字符串时仍可拆开）。
4. 文案提示改为「可用中英文逗号/分号分隔」；placeholder 示例使用中文逗号。
5. 公网 SPA：`runall-lifecycle.sh build`（build + collectstatic + manifest）。

## 验证

```bash
cd taskFE/app && npm test -- --run src/utils/featureParamsModelValidation.test.js
# 期望含 Chinese comma 用例通过

# 浏览器：在支持模型列表输入 gpt-4.1，gpt-4.1-mini 后选供应商，
# 智能体模型 datalist 应出现两个候选，而非一条「gpt-4.1，gpt-4.1-mini」
```

## 预防

- 用户可见「列表输入」分隔逻辑须集中在一处工具函数，禁止各页面复制 `/[\n, ]/`。
- 中文输入场景默认把 `，` / `；` 视为分隔符；改分隔规则时前后端同测。
