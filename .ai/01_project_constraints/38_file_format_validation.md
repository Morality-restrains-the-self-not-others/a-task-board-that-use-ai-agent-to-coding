# 文件修改后格式验证（元规则）

- 版本：1.0.0
- 创建日期：2026-07-28
- 最后修改：2026-07-28
- 维护者：ljy

## 规则分类

### 核心规则（一级分类）
> 修改任何文件后必须使用对应工具验证格式有效性，缺工具则安装；影响代码质量和可维护性，必须严格遵守

#### 代码质量保障（二级分类）

##### 文件修改后格式验证（三级分类）
- 描述：修改任何文件后，必须在提交前使用对应格式验证工具确认文件格式有效。若缺乏验证工具，必须先安装再验证。覆盖 Go、Python、JavaScript/TypeScript/Vue、YAML、JSON、Shell、Dockerfile、Markdown、SQL、HTML/CSS 等全部文件类型
- 适用场景：所有文件的新建、修改、批量修改；Agent 自动修改与人工修改均适用
- 优先级：高（核心规则）
- 规则类型：禁止忽略

#### 验证工具清单

| 文件类型 | 格式化验证 | 静态分析/Lint | 内建 Fallback |
|---------|-----------|---------------|---------------|
| `.go` | `gofmt -d <file>` | `go vet ./...` | gofmt + go vet（go 自带） |
| `.py` | `python3 -m py_compile <file>` | `ruff check <file>` | `python3 -m py_compile` |
| `.js` `.ts` `.tsx` `.jsx` | `eslint --fix-dry-run <file>` | `eslint <file>` | `node --check <file>` |
| `.vue` | `eslint --fix-dry-run <file>` | `eslint <file>` | `node --check <file>` |
| `.yaml` `.yml` | `python3 -c "import yaml; yaml.safe_load(open('<file>'))"` | `yamllint <file>` | `python3 yaml.safe_load` |
| `.json` | `jq . <file> > /dev/null` | `jq . <file> > /dev/null` | `python3 -m json.tool` |
| `.sh` `.bash` | `bash -n <file>` | `shellcheck <file>` | `bash -n` |
| `Dockerfile*` | `hadolint <file>` | `hadolint <file>` | 无（需安装 hadolint） |
| `.md` | `markdownlint <file>` | `markdownlint <file>` | 无（需安装 markdownlint） |
| `.sql` | `sqlfluff lint <file>` | `sqlfluff lint <file>` | 无（需安装 sqlfluff） |
| `.html` `.css` | `prettier --check <file>` | `prettier --check <file>` | 无（需安装 prettier） |

#### 安装命令

网络可用时，按需安装缺失工具：

```bash
# Python 工具
pip install ruff --break-system-packages
pip install yamllint --break-system-packages
pip install sqlfluff --break-system-packages

# Node.js 工具（全局）
npm install -g eslint eslint-plugin-vue
npm install -g prettier
npm install -g markdownlint-cli

# Shell 工具
sudo apt install shellcheck
# 或 snap install shellcheck

# Docker 工具
docker pull hadolint/hadolint
```

#### 执行流程

1. **修改文件**
2. **根据文件类型选择对应工具**（参见对照表）
3. **执行格式化验证**（语法/格式合法性检查）
4. 若格式化验证失败 → 修复 → 回到步骤 3
5. **执行静态分析/Lint**（代码质量检查）
6. 若 lint 失败 → 修复 → 回到步骤 5
7. **两项均通过 → 可以提交**

#### JSON/YAML 特别注意事项

- JSON 文件（`conf/port_config.json`、`.mcp.json`、`package.json` 等）格式错误会导致服务启动失败，禁止使用 echo/手工拼接构造
- YAML 文件（`conf/base.yaml`、`docker-compose.yml`、CI workflow 等）缩进错误难以调试，严禁 Tab 缩进，必须通过 `yaml.safe_load` 验证
- 所有配置文件的修改必须在下一轮操作中立即验证，不得累计多轮后批量检查

## 规则冲突处理

- 当与其他规则冲突时：本规则要求的格式验证为提交前置条件，不得被其他规则覆盖
- 核心规则 > 最佳实践 > 风格指南
- 本规则与 `.cursor/rules/file-format-validation.mdc` 为同一规则的不同载体，以 Cursor 规则为执行触发，以本文为详细参考

## 变更日志

- 2026-07-28：版本 1.0.0 - 初始创建；定义 10 种文件类型的格式验证工具对照表；确立「缺工具则安装」原则
