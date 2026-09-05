// @ts-check
/**
 * 产物/源码新鲜度校验（OPT-20260824-056）。
 *
 * 背景：E2E 调试最隐蔽根因是 :4000 产物服务承载旧构建（public/html → 旧 release），
 * UI 断言打旧产物导致「产物与源码漂移」失败并触发 worker 重启级联。
 * 本用例在 E2E 套件中固化「先校新鲜度」：产物 build-time 早于源码最新 mtime 时默认
 * 仅告警（不阻断）；设置 REQUIRE_FRESH=1 时升级为失败。
 */
import { test, expect } from '@playwright/test';
import {
  checkProdBuildFreshness,
  defaultSourceDir,
} from './helpers/prodBuildFreshness.js';

test('产物 build-time 不早于源码最新 mtime（漂移仅告警，REQUIRE_FRESH=1 时失败）', async ({
  request,
  baseURL,
}) => {
  const report = await checkProdBuildFreshness({
    baseUrl: baseURL,
    sourceDir: defaultSourceDir(),
    fetchFn: (url) => request.get(url),
    logger: console,
  });

  if (process.env.REQUIRE_FRESH === '1') {
    expect(report.fresh, report.reason || '产物新鲜度未知').toBe(true);
  } else {
    // 默认仅提示；产物不可达（服务未起）也算通过，避免拖垮其他用例
    expect(report).toBeTruthy();
  }
});
