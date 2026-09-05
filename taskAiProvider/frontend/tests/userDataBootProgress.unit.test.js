import test from "node:test";
import assert from "node:assert/strict";
import { generateTemplateContent } from "../src/composables/useUserDataScriptGenerator.js";

test("Linux 模板含 boot-progress 逐步上报与云前缀占位符", () => {
  const content = generateTemplateContent({
    os_type: "ubuntu_22_04",
    content: "",
    auto_verify_script: "",
    userdata_variables: [
      // 即使误填真实值，生成器也必须强制占位符
      { name: "ACCESS_TOKEN", value: "tok_should_not_appear" },
      { name: "TASK_API_ENDPOINT", value: "https://evil.example/cloud" },
      { name: "CONTAINER_IMAGE", value: "registry.example/should-not-bake:latest" },
    ],
    container_variables: [
      { name: "RUNTIME", value: "docker" },
      { name: "CONTAINER_NAME", value: "real-name-should-not-bake" },
    ],
  });
  assert.match(content, /report_progress/);
  assert.match(content, /server-container-token\/boot-progress\//);
  assert.match(content, /__TASK2APP_TASK_CLOUD_PREFIX__/);
  assert.match(content, /export ACCESS_TOKEN='__TASK2APP_ACCESS_TOKEN__'/);
  assert.match(content, /export TASK_API_ENDPOINT='__TASK2APP_TASK_CLOUD_PREFIX__'/);
  assert.match(content, /CONTAINER_IMAGE='__TASK2APP_CONTAINER_IMAGE__'/);
  assert.match(content, /export CONTAINER_NAME='__TASK2APP_CONTAINER_NAME__'/);
  assert.match(content, /export COMMENT_ID='__TASK2APP_COMMENT_ID__'/);
  assert.match(content, /export TRACE_ID='__TASK2APP_TRACE_ID__'/);
  assert.doesNotMatch(content, /tok_should_not_appear/);
  assert.doesNotMatch(content, /evil\.example/);
  assert.doesNotMatch(content, /should-not-bake/);
  assert.doesNotMatch(content, /real-name-should-not-bake/);
  assert.match(content, /report_progress 5 /);
  assert.match(content, /report_progress 95 /);
  assert.match(content, /comment_id/);
  assert.match(content, /container_name/);
  assert.match(content, /trace_id/);
  assert.match(content, /X-Trace-Id/);
});

test("Linux report_progress _extra 赋值双引号须闭合（否则 comment_id JSON 损坏）", () => {
  const content = generateTemplateContent({
    os_type: "ubuntu_22_04",
    content: "",
    auto_verify_script: "",
    userdata_variables: [],
    container_variables: [],
  });
  // 闭合形态：...\"${COMMENT_ID}\""  — 末尾字面 \" 后再跟赋值闭合 "
  assert.match(
    content,
    /_extra="\$\{_extra\},\\"comment_id\\":\\"\$\{COMMENT_ID\}\\""/,
  );
  assert.match(
    content,
    /_extra="\$\{_extra\},\\"container_name\\":\\"\$\{CONTAINER_NAME\}\\""/,
  );
  assert.match(
    content,
    /_extra="\$\{_extra\},\\"trace_id\\":\\"\$\{TRACE_ID\}\\""/,
  );
});

test("Linux 健康检查使用 shell CONTAINER_NAME 而非未定义的 containerName", () => {
  const content = generateTemplateContent({
    os_type: "ubuntu_22_04",
    content: "",
    auto_verify_script: "",
    userdata_variables: [],
    container_variables: [],
  });
  assert.match(
    content,
    /inspect -f '\{\{\.State\.Status\}\}' "\$\{CONTAINER_NAME\}"/,
  );
  assert.match(content, /top "\$\{CONTAINER_NAME\}"/);
  assert.doesNotMatch(
    content,
    /inspect -f '\{\{\.State\.Status\}\}' \$\{containerName\}/,
  );
  assert.doesNotMatch(content, /top \$\{containerName\}/);
});

test("Linux Docker 安装优先国内镜像并回退官方源（避免 download.docker.com TLS reset）", () => {
  const content = generateTemplateContent({
    os_type: "ubuntu_22_04",
    content: "",
    auto_verify_script: "",
    userdata_variables: [],
    container_variables: [{ name: "RUNTIME", value: "docker" }],
  });
  assert.match(content, /mirrors\.aliyun\.com\/docker-ce\/linux\/\$\{OS_FAMILY\}/);
  assert.match(
    content,
    /mirrors\.tuna\.tsinghua\.edu\.cn\/docker-ce\/linux\/\$\{OS_FAMILY\}/,
  );
  assert.match(content, /download\.docker\.com\/linux\/\$\{OS_FAMILY\}/);
  assert.match(content, /检测到已安装 docker，跳过 Docker Engine 安装/);
  assert.match(
    content,
    /错误: Docker CE 安装失败（GPG\/仓库均不可达，含 download\.docker\.com）/,
  );
  // 禁止仅依赖官方源的旧单点 curl|gpg（无镜像循环）
  assert.doesNotMatch(
    content,
    /curl -fsSL https:\/\/download\.docker\.com\/linux\/ubuntu\/gpg \| gpg --dearmor/,
  );
  // 未使用的 {REGISTRY_URL} 易误导排障，生成器不得再输出
  assert.doesNotMatch(content, /REGISTRY_URL=/);
  assert.doesNotMatch(content, /\{REGISTRY_URL\}/);
});

test("Windows 模板含 Report-Progress 与 boot-progress", () => {
  const content = generateTemplateContent({
    os_type: "windows_server_2022",
    content: "",
    auto_verify_script: "",
    userdata_variables: [],
    container_variables: [],
  });
  assert.match(content, /Report-Progress/);
  assert.match(content, /server-container-token\/boot-progress\//);
  assert.match(content, /__TASK2APP_TASK_CLOUD_PREFIX__/);
  assert.match(content, /\$env:COMMENT_ID = '__TASK2APP_COMMENT_ID__'/);
  assert.match(content, /\$env:TRACE_ID = '__TASK2APP_TRACE_ID__'/);
  assert.match(content, /\$env:CONTAINER_NAME = '__TASK2APP_CONTAINER_NAME__'/);
  assert.match(content, /comment_id/);
  assert.match(content, /trace_id/);
});
