// @ts-check
/**
 * 校验 GET /api/git-oauth/github-app-start/ 返回的 authorize_url 指向 gitOauth，
 * 且经 gitOauth 跳转后的 GitHub authorize URL 中 redirect_uri 与 conf/git-oauth（聚合）
 * 的 gitOauth.redirect_uri 一致。
 *
 * 需：前端 Vite + Django + gitOauth 已启动；PLAYWRIGHT_TEST_EMAIL / PLAYWRIGHT_TEST_PASSWORD
 */
import { test, expect } from '@playwright/test';
import { playwrightLoginWithLegalAccept } from './playwrightLogin.js';
import { fileURLToPath } from 'url';
import { loadPortConfig } from '../helpers/loadConfYaml.mjs';

const readGithubOauthPortConfig = () => {
  try {
    const raw = loadPortConfig();
    const gitOauthRaw = raw?.gitOauth;
    const g = Array.isArray(gitOauthRaw)
      ? gitOauthRaw.find((item) => item && typeof item === 'object')
      : undefined;
    const redirectUri =
      g && typeof g === 'object' && typeof g.redirect_uri === 'string' ? g.redirect_uri.trim() : '';
    // OPT-20260806-053: Django 退役，django.gitoauth 字段已移除 →
    // 迁移到 _addressing.addresses.gitoauth（conf/base.yaml subdomains.gitoauth）
    const addressing = raw?._addressing?.addresses;
    const gitoauthHost =
      addressing && typeof addressing.gitoauth === 'string' ? addressing.gitoauth.trim() : '';
    const scheme = addressing?.scheme || 'https';
    const gitoauthBase = gitoauthHost ? `${scheme}://${gitoauthHost}` : '';
    return {
      gitoauthBase: gitoauthBase.replace(/\/$/, ''),
      redirectUri,
    };
  } catch {
    return { gitoauthBase: '', redirectUri: '' };
  }
};

test('github/start 的 authorize_url 经 gitOauth 且 redirect_uri 与 port_config 一致', async ({
  page,
}) => {
  const email = process.env.PLAYWRIGHT_TEST_EMAIL || '';
  const password = process.env.PLAYWRIGHT_TEST_PASSWORD || '';
  test.skip(!email || !password, '设置 PLAYWRIGHT_TEST_EMAIL 与 PLAYWRIGHT_TEST_PASSWORD');

  const { gitoauthBase, redirectUri } = readGithubOauthPortConfig();
  test.skip(!gitoauthBase, `无法从 conf 读取 addressing.gitoauth`);
  test.skip(!redirectUri, `无法从 conf 读取 gitOauth.redirect_uri`);
  expect(redirectUri).toContain('/api/accounts/github/oauth/callback');

  await playwrightLoginWithLegalAccept(page, { email, password });

  const returnKey = Array.from({ length: 32 }, () =>
    Math.floor(Math.random() * 16).toString(16),
  ).join('');
  const nextPath = '/profile/git-site-oauth/';
  const startUrl = `/api/git-oauth/github-app-start/?next=${encodeURIComponent(nextPath)}&return_key=${encodeURIComponent(returnKey)}`;

  const startRes = await page.request.get(startUrl, {
    headers: { Accept: 'application/json' },
  });
  expect(startRes.ok(), `start 应 200，实际 ${startRes.status()} ${await startRes.text()}`).toBeTruthy();

  const body = await startRes.json();
  expect(body.authorize_url).toBeTruthy();

  const gitoauthStart = new URL(String(body.authorize_url));
  expect(gitoauthStart.pathname).toBe('/api/git-oauth/github-start/');
  expect(gitoauthStart.searchParams.get('token')).toBeTruthy();

  const gitoauthOrigin = new URL(gitoauthBase).origin;
  expect(gitoauthStart.origin).toBe(gitoauthOrigin);

  const hopRes = await page.request.get(body.authorize_url, { maxRedirects: 0 });
  expect(
    hopRes.status() === 302 || hopRes.status() === 301,
    `gitOauth start 应重定向到 GitHub，实际 ${hopRes.status()} ${await hopRes.text()}`,
  ).toBeTruthy();
  const loc = hopRes.headers()['location'];
  expect(loc).toBeTruthy();
  const githubAuthorize = new URL(String(loc));
  expect(githubAuthorize.hostname).toBe('github.com');

  const redirectUriParam = githubAuthorize.searchParams.get('redirect_uri');
  expect(redirectUriParam).toBe(redirectUri);
});
