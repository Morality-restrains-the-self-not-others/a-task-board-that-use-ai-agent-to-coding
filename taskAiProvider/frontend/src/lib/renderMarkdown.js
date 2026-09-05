/**
 * 轻量 Markdown → HTML 渲染器（面向 SSOT 文档：标题/段落/表格/代码/引用/列表）。
 * 不引入第三方依赖；输出经 HTML 转义，避免文档内代码片段被当作标签执行。
 */

function escapeHtml(text) {
  return String(text)
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

/** 行内解析：`code` / **bold** / [text](url) */
function renderInline(raw) {
  let text = String(raw ?? "");
  // 先转义，再对转义后的文本做结构化替换
  text = escapeHtml(text);
  text = text.replace(/`([^`]+)`/g, (_m, code) => `<code>${code}</code>`);
  text = text.replace(/\*\*([^*]+)\*\*/g, (_m, bold) => `<strong>${bold}</strong>`);
  text = text.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (_m, label, href) => {
    const safe = /^(https?:|#|\/)/.test(href) ? href : "#";
    return `<a href="${safe}" target="_blank" rel="noopener noreferrer">${label}</a>`;
  });
  return text;
}

function renderTable(lines, i) {
  const headerRow = lines[i];
  const alignRow = lines[i + 1];
  if (!alignRow || !/^\s*\|[\s:\-|]+\|\s*$/.test(alignRow)) {
    return null;
  }
  const split = (row) =>
    row
      .trim()
      .replace(/^\|/, "")
      .replace(/\|$/, "")
      .split("|")
      .map((cell) => cell.trim());

  const headers = split(headerRow);
  let html = "<table><thead><tr>";
  for (const h of headers) html += `<th>${renderInline(h)}</th>`;
  html += "</tr></thead><tbody>";

  let j = i + 2;
  while (j < lines.length && /^\s*\|/.test(lines[j])) {
    const cells = split(lines[j]);
    html += "<tr>";
    for (let c = 0; c < headers.length; c += 1) {
      html += `<td>${renderInline(cells[c] ?? "")}</td>`;
    }
    html += "</tr>";
    j += 1;
  }
  html += "</tbody></table>";
  return { html, next: j };
}

/**
 * @param {string} markdown
 * @returns {string} HTML 片段
 */
export function renderMarkdown(markdown) {
  const lines = String(markdown ?? "").split(/\r?\n/);
  const out = [];
  let i = 0;
  let inFence = false;
  let fenceBuf = [];

  const flushFence = () => {
    if (fenceBuf.length) {
      out.push(`<pre><code>${escapeHtml(fenceBuf.join("\n"))}</code></pre>`);
      fenceBuf = [];
    }
  };

  while (i < lines.length) {
    const line = lines[i];

    if (/^\s*```/.test(line)) {
      if (inFence) {
        inFence = false;
        flushFence();
      } else {
        flushFence();
        inFence = true;
      }
      i += 1;
      continue;
    }
    if (inFence) {
      fenceBuf.push(line);
      i += 1;
      continue;
    }

    const heading = /^(#{1,4})\s+(.*)$/.exec(line);
    if (heading) {
      const level = heading[1].length;
      out.push(`<h${level}>${renderInline(heading[2])}</h${level}>`);
      i += 1;
      continue;
    }

    if (/^\s*\|/.test(line)) {
      const table = renderTable(lines, i);
      if (table) {
        out.push(table.html);
        i = table.next;
        continue;
      }
    }

    if (/^>\s?/.test(line)) {
      const quote = [];
      while (i < lines.length && /^>\s?/.test(lines[i])) {
        quote.push(lines[i].replace(/^>\s?/, ""));
        i += 1;
      }
      out.push(`<blockquote>${renderInline(quote.join(" "))}</blockquote>`);
      continue;
    }

    if (/^\s*[-*]\s+/.test(line)) {
      out.push("<ul>");
      while (i < lines.length && /^\s*[-*]\s+/.test(lines[i])) {
        out.push(`<li>${renderInline(lines[i].replace(/^\s*[-*]\s+/, ""))}</li>`);
        i += 1;
      }
      out.push("</ul>");
      continue;
    }

    if (/^\s*\d+\.\s+/.test(line)) {
      out.push("<ol>");
      while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i])) {
        out.push(`<li>${renderInline(lines[i].replace(/^\s*\d+\.\s+/, ""))}</li>`);
        i += 1;
      }
      out.push("</ol>");
      continue;
    }

    // 空行或连续段落
    const para = [];
    while (i < lines.length && lines[i].trim() !== "" && !/^\s*(\||```|#|>|[-*]|\d+\.)/.test(lines[i])) {
      para.push(lines[i]);
      i += 1;
    }
    if (para.length) {
      out.push(`<p>${renderInline(para.join(" "))}</p>`);
    } else {
      i += 1;
    }
  }

  flushFence();
  return out.join("\n");
}
