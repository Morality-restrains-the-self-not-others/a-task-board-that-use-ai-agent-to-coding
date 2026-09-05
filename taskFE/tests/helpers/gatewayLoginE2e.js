// @ts-check
/**
 * 网关 API 登录（与 Login.vue 一致：password 为明文）。
 *
 * OPT-20260812-022：taskAuth handleLogin 对明文密码直接做 bcrypt 比对
 * （bcrypt 仅支持 ≤72 字节；Django pbkdf2_sha256 产物恒 >72 字节必失败）。
 * 此前此处误将 password 做 generateDjangoPasswordHash 后再发送，导致所有
 * 经 loginViaGatewayApi 的 E2E 恒返回「用户名或密码错误」。改为直发明文。
 */
import { clientReachableHost, loadPortConfig } from '../../helpers/loadConfYaml.mjs';

/**
 * @returns {{ siteOrigin: string; gatewayOrigin: string }}
 */
export function readE2eOrigins() {
  const portConfig = loadPortConfig();
  const siteOrigin = (
    process.env.SITE_BASE ||
    process.env.PLAYWRIGHT_SITE_ORIGIN ||
    process.env.BASE_URL ||
    portConfig.vue?.publicBaseUrl ||
    `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
  ).replace(/\/$/, '');

  const gatewayOrigin = (
    process.env.PLAYWRIGHT_GATEWAY_ORIGIN ||
    process.env.GATEWAY_URL ||
    portConfig.vue?.apiBaseUrl ||
    // Django saas-backend 已退役（2026-07-30）；网关为 APISIX（18081），
    // host 与前端同机（OPT-20260806-053：django.host 引用迁移）
    `http://${clientReachableHost(portConfig.vue?.host || '127.0.0.1')}:18081`
  ).replace(/\/$/, '');

  return { siteOrigin, gatewayOrigin };
}

/**
 * 通过网关 POST /api/auth/ 登录（password 为明文，与 useLoginSubmit.js 一致），
 * cookies 写入 page 上下文。
 *
 * @param {import('@playwright/test').Page} page
 * @param {{ email: string; password: string; siteOrigin?: string; gatewayOrigin?: string }} options
 */
export async function loginViaGatewayApi(page, options) {
  const defaults = readE2eOrigins();
  const siteOrigin = (options.siteOrigin || defaults.siteOrigin).replace(/\/$/, '');
  const gatewayOrigin = (options.gatewayOrigin || defaults.gatewayOrigin).replace(/\/$/, '');
  const email = String(options.email || '').trim();
  const plainPassword = String(options.password || '');

  if (!email || !plainPassword) {
    throw new Error('loginViaGatewayApi 缺少 email 或 password');
  }

  const originHeader = { Origin: siteOrigin, Accept: 'application/json' };

  const privacyRes = await page.request.get(`${gatewayOrigin}/api/privacy-policy/public/current/`, {
    headers: originHeader,
    timeout: 30000,
  });
  const licenseRes = await page.request.get(`${gatewayOrigin}/api/license-agreement/public/current/`, {
    headers: originHeader,
    timeout: 30000,
  });

  if (!privacyRes.ok() || !licenseRes.ok()) {
    throw new Error(
      `无法加载隐私/许可条款: privacy=${privacyRes.status()} license=${licenseRes.status()}`,
    );
  }

  const privacy = await privacyRes.json();
  const license = await licenseRes.json();

  const loginRes = await page.request.post(`${gatewayOrigin}/api/auth/`, {
    headers: {
      ...originHeader,
      'Content-Type': 'application/json',
    },
    data: {
      username: email,
      password: plainPassword,
      rememberMe: true,
      accepted_privacy_policy_id: String(privacy?.id || ''),
      accepted_license_agreement_id: String(license?.id || ''),
    },
    timeout: 30000,
  });

  if (!loginRes.ok()) {
    const body = await loginRes.text().catch(() => '');
    throw new Error(`网关登录失败: HTTP ${loginRes.status()} ${body.slice(0, 300)}`);
  }

  const loginJson = await loginRes.json();
  const token = String(loginJson?.token || '').trim();
  const userId = String(loginJson?.user?.id || loginJson?.user_id || '').trim();

  // Auth JSON 登录不再直接 Set-Cookie；须 activate-session 落 HttpOnly 会话。
  // OPT-20260904-006：优先经 siteOrigin 激活，使 Set-Cookie（Domain=daydaymoney.com）
  // 能由浏览器/请求上下文自然落在与前端一致域（跨子域 OAuth 回调亦生效）。此前
  // 本机 :4000 的 /api 代理会把同一物理主机的 caller 解析成 loopback 127.0.0.1，
  // 与网关 docker-bridge 视角 172.25.0.1 不一致 → taskAuth 误报 500 "db error"，
  // 只能绕行 gatewayOrigin。taskAuth 侧已对 loopback↔bridge 同机视图放行；
  // 此处仍保留 gatewayOrigin 兜底 + 镜像 host-only cookie（对 127.0.0.1 域无效）。
  /** @type {string[]} */
  let setCookieLines = [];
  if (token) {
    const activateOptions = {
      headers: {
        Authorization: `Token ${token}`,
        Origin: siteOrigin,
        Accept: 'application/json',
        'Content-Type': 'application/json',
      },
      data: userId ? { user_id: userId } : {},
      timeout: 30000,
    };
    const activateOrigins = [...new Set([siteOrigin, gatewayOrigin])];
    let activateRes = null;
    let lastErr = '';
    for (const origin of activateOrigins) {
      activateRes = await page.request.post(
        `${origin}/api/accounts/users/activate-session/`,
        activateOptions,
      );
      if (activateRes.ok()) break;
      lastErr = `HTTP ${activateRes.status()} ${(await activateRes.text().catch(() => '')).slice(0, 200)}`;
      activateRes = null;
    }
    if (!activateRes) {
      throw new Error(`activate-session 失败: ${lastErr}`);
    }
    setCookieLines = activateRes
      .headersArray()
      .filter((h) => h.name.toLowerCase() === 'set-cookie')
      .map((h) => h.value);
  }

  /** @type {import('@playwright/test').Cookie[]} */
  const mirror = [];
  for (const line of setCookieLines) {
    const nameVal = line.split(';')[0] || '';
    const eq = nameVal.indexOf('=');
    if (eq <= 0) continue;
    const name = nameVal.slice(0, eq).trim();
    const value = nameVal.slice(eq + 1).trim();
    if (!name || !value) continue; // 跳过 Max-Age=0 清 cookie
    if (!['userId', 'token', 'sessionid', 'csrftoken'].includes(name)) continue;
    mirror.push({
      name,
      value: decodeURIComponent(value),
      url: `${siteOrigin}/`,
      httpOnly: /;\s*HttpOnly/i.test(line),
    });
  }
  if (userId && !mirror.some((c) => c.name === 'userId')) {
    mirror.push({ name: 'userId', value: userId, url: `${siteOrigin}/` });
  }
  if (mirror.length) {
    await page.context().addCookies(mirror);
  }
  if (userId) {
    await page.addInitScript((id) => {
      try {
        localStorage.setItem('currentUserId', id);
      } catch {
        /* ignore */
      }
    }, userId);
  }

  return loginJson;
}

/**
 * 登录后导航到指定路径；若仍被重定向到登录页则抛错。
 *
 * @param {import('@playwright/test').Page} page
 * @param {string} targetUrl
 */
export async function gotoAuthenticatedPath(page, targetUrl) {
  await page.goto(targetUrl, { waitUntil: 'domcontentloaded', timeout: 60_000 });
  // SPA 可能先渲染再 client redirect 到登录页
  await page.waitForTimeout(500);
  if (page.url().includes('/auth/login')) {
    throw new Error(`登录后会话未生效，仍被重定向到登录页: ${page.url()}`);
  }
}
