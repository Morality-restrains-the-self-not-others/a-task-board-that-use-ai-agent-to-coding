# 登录页面 `Unexpected token 'export'` 错误

## 问题描述
用户访问 `http://localhost:8000/auth/login/` 页面时，浏览器控制台出现 `Unexpected token 'export'` 错误，导致页面无法正常加载。

## 错误信息

## 环境信息
- 操作系统：macOS
- 前端框架：Vue 3
- 构建工具：Vite
- 后端框架：Django 4.2.26

## 原因分析
1. **问题定位**：使用 playwright 测试登录页面，捕获网络请求和响应，发现错误发生在加载 `http://localhost:8000/static/js/utils.js` 文件后。
2. **文件内容检查**：检查 `utils.js` 文件内容，发现该文件包含 `export function getCookie(name) { ... }` 这一行，使用了 ES 模块的导出语法。
3. **文件来源分析**：在 `settings.py` 文件中，`STATICFILES_DIRS` 配置了多个目录，其中 `Saas_project/static/` 排在第一位，优先于 `Saas_project/frontend/static/` 目录。
4. **根因确认**：浏览器尝试以非模块方式加载 `static/js/utils.js` 文件，而该文件使用了 ES 模块的 `export` 语法，导致语法错误。

## 解决方案
1. **修改文件**：删除 `Saas_project/static/js/utils.js` 文件中的 `export` 关键字，使它成为一个普通的 JavaScript 文件。
2. **验证修改**：重新测试登录页面，确认 `Unexpected token 'export'` 错误已经消失。

## 验证结果
1. **错误消失**：重新测试登录页面，浏览器控制台不再出现 `Unexpected token 'export'` 错误。
2. **页面正常加载**：登录页面可以正常打开，登录表单被找到，路由导航正常工作。
3. **功能正常**：用户可以正常输入登录信息，尝试登录操作。

## 预防措施
1. **文件命名规范**：确保静态文件目录中的 JavaScript 文件不使用 ES 模块语法，或者使用 `.mjs` 扩展名明确标识为 ES 模块。
2. **目录结构管理**：定期检查静态文件目录结构，确保没有重复的文件导致冲突。
3. **构建配置**：确保 Vite 构建配置正确，避免将 ES 模块语法的文件输出到静态文件目录。
4. **测试流程**：在部署前，使用 playwright 等工具测试关键页面，确保没有语法错误。

## 技术教训
1. **静态文件处理**：了解 Django 和 Vite 如何处理静态文件，特别是多个静态文件目录的优先级顺序。
2. **模块系统**：理解 ES 模块和 CommonJS 模块的区别，以及浏览器如何加载不同类型的 JavaScript 文件。
3. **调试技巧**：使用网络请求和响应捕获工具，定位问题的具体来源。
4. **经验记录**：及时记录解决的问题和解决方案，为团队积累经验。
