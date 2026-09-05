import assert from 'node:assert/strict';
import { describe, it } from 'node:test';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

// OPT-20260812-056：厂商门户关联写入强制 CSI 地域对齐。
// 后端 UpsertAssociation/批量写入已以 CSI.region 为准；本测试锁定 UI 侧不提供
// 独立 region 下拉——region 只经 prop 传入（只读），保存载荷取自 props.region?.id。
const root = dirname(fileURLToPath(import.meta.url));
const modalSrc = readFileSync(join(root, '../src/components/RegionEnvModal.vue'), 'utf8');

describe('RegionEnvModal CSI 地域只读对齐', () => {
  it('不包含独立 region 下拉（region 为 prop，只读）', () => {
    // 模板层：不允许出现绑定到 region 的 <select>（v-model="form.region" 或 v-model="region"）
    const regionSelectPatterns = [
      /v-model=["']form\.region["']/,
      /v-model=["']region["']/,
      /:options=.*region.*select/i,
      /id=["']region-select["']/i,
    ];
    for (const re of regionSelectPatterns) {
      assert.equal(re.test(modalSrc), false, `unexpected region select pattern: ${re}`);
    }
    // 确认模板中存在 region 只读展示（region?.name）
    assert.match(modalSrc, /region\?\.name/);
  });

  it('保存载荷 region 取自 props.region?.id（CSI 地域），而非表单字段', () => {
    assert.match(modalSrc, /regionId:\s*props\.region\?\.id/);
    // 不允许保存时从 form 取 region
    assert.equal(/regionId:\s*form\.region/.test(modalSrc), false);
  });

  it('region 声明为 prop 且带 CSI 语义注释', () => {
    assert.match(modalSrc, /region:\s*\{\s*type:\s*Object,\s*default:\s*null\s*\}/);
  });
});
