import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { fileURLToPath } from "node:url";
import { renderMarkdown } from "../src/lib/renderMarkdown.js";

const here = path.dirname(fileURLToPath(import.meta.url));

describe("renderMarkdown", () => {
  it("renders a markdown table with headers and rows", () => {
    const md = `## 1. 前缀与鉴权\n\n| 变量 | 说明 |\n|------|------|\n| \`ACCESS_TOKEN\` | 容器访问令牌 |\n| \`PORT\` | 容器端口 |\n`;
    const html = renderMarkdown(md);

    assert.match(html, /<h2>1\. 前缀与鉴权<\/h2>/);
    assert.match(html, /<table>/);
    assert.match(html, /<th>变量<\/th>/);
    assert.match(html, /<th>说明<\/th>/);
    assert.match(html, /<td><code>ACCESS_TOKEN<\/code><\/td>/);
    assert.match(html, /<td>容器访问令牌<\/td>/);
  });

  it("renders the SSOT doc with tables (table title visible)", () => {
    const mdPath = path.join(
      here,
      "../../../docs/skills/saas-container/saas-machine-container.md",
    );
    const md = readFileSync(mdPath, "utf8");
    const html = renderMarkdown(md);

    // 表格标题「1. 前缀与鉴权」及其表头可见
    assert.match(html, /<h2>1\. 前缀与鉴权<\/h2>/);
    assert.match(html, /<th>变量<\/th>/);
    assert.match(html, /<th>说明<\/th>/);
    // taskCredentialService 接口表头
    assert.match(html, /<th>action<\/th>/);
    assert.match(html, /<th>用途<\/th>/);
    assert.ok((html.match(/<table>/g) || []).length >= 3, "SSOT 文档应渲染多个表格");
    assert.match(html, /\/api\/git-oauth\/merge-request-status\/tenant_id\/\{tid\}\//);
    assert.match(html, /\/api\/git-oauth\/merge-request-merge\/tenant_id\/\{tid\}\//);
    assert.match(html, /\/api\/tenant_id\/\{tid\}\/workspaceId\/\{wid\}\/tasks\/\{taskId\}\/comments\/\{parent_comment_id\}\//);
  });

  it("escapes HTML in inline code and escapes code fence content", () => {
    const md = "## 标题\n\n`<script>alert(1)</script>`\n\n```\n<b>raw</b>\n```\n";
    const html = renderMarkdown(md);
    assert.match(html, /<code>&lt;script&gt;alert\(1\)&lt;\/script&gt;<\/code>/);
    assert.doesNotMatch(html, /<script>alert/);
    assert.match(html, /<pre><code>&lt;b&gt;raw&lt;\/b&gt;<\/code><\/pre>/);
  });

  it("renders blockquote and list", () => {
    const md = "> 所有接口均为 POST JSON\n\n- exchange-refresh\n- refresh-access\n";
    const html = renderMarkdown(md);
    assert.match(html, /<blockquote>所有接口均为 POST JSON<\/blockquote>/);
    assert.match(html, /<ul>\s*<li>exchange-refresh<\/li>\s*<li>refresh-access<\/li>\s*<\/ul>/);
  });
});
