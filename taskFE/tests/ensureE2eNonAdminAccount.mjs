/**
 * 幂等确保 E2E 非超管账号存在且已激活。
 * 默认：e2e.nonadmin.sysadmin.redirect@ljytest.com
 */
import { execFileSync } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const REPO_ROOT = path.resolve(__dirname, '../../../../');
const FRONT_APP = path.join(REPO_ROOT, 'taskFE/app');
const AUTH_DB = process.env.TASKAUTH_DB_PATH || path.join(REPO_ROOT, 'db/task-auth/auth.sqlite3');

const SITE = (process.env.PLAYWRIGHT_SITE_ORIGIN || 'https://www.daydaymoney.com').replace(/\/$/, '');
const EMAIL =
  process.env.E2E_NONADMIN_EMAIL || 'e2e.nonadmin.sysadmin.redirect@ljytest.com';
const PASSWORD = process.env.E2E_NONADMIN_PASSWORD || 'E2eNonAdmin!8677';
const ACCESS_CODE = process.env.REGISTER_ACCESS_CODE || 'u824976301710503936';

function sql(query) {
  return execFileSync('sqlite3', [AUTH_DB, query], { encoding: 'utf8' }).trim();
}

async function postJson(url, body) {
  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
    body: JSON.stringify(body),
  });
  const text = await res.text();
  let json = null;
  try {
    json = JSON.parse(text);
  } catch {
    /* ignore */
  }
  return { status: res.status, json, text };
}

/** 与 Register.vue / Login.vue 一致的前端哈希（identifier=email） */
function hashPasswordForRegister(plain, email) {
  const code = `
    import { generateDjangoPasswordHash } from './src/components/PasswordHasher.js';
    const h = await generateDjangoPasswordHash(${JSON.stringify(plain)}, ${JSON.stringify(email)});
    process.stdout.write(h);
  `;
  return execFileSync(process.execPath, ['--input-type=module', '-e', code], {
    encoding: 'utf8',
    cwd: FRONT_APP,
  }).trim();
}

export async function ensureE2eNonAdminAccount() {
  const existing = sql(
    `SELECT object_id || '|' || COALESCE(is_verified,0) FROM accounts_login_method WHERE identifier='${EMAIL.replace(/'/g, "''")}';`,
  );
  if (!existing) {
    const passwordHash = hashPasswordForRegister(PASSWORD, EMAIL);
    const reg = await postJson(`${SITE}/api/accounts/users/email_register/`, {
      email: EMAIL,
      password: passwordHash,
      username: 'e2e_nonadmin',
      accessCode: ACCESS_CODE,
    });
    if (reg.status >= 400 && !(reg.json && reg.json.user_existed)) {
      throw new Error(`注册失败 ${reg.status}: ${reg.text.slice(0, 300)}`);
    }
    console.log('[ensure-nonadmin] registered', EMAIL, reg.status);
  }

  const row = sql(
    `SELECT object_id || '|' || COALESCE(is_verified,0) || '|' || COALESCE(activation_token,'') FROM accounts_login_method WHERE identifier='${EMAIL.replace(/'/g, "''")}';`,
  );
  if (!row) throw new Error(`注册后仍找不到登录方式: ${EMAIL}`);
  const [userId, verified, token] = row.split('|');
  if (verified !== '1') {
    if (!token) throw new Error(`账号未激活且无 activation_token: ${EMAIL}`);
    const act = await postJson(`${SITE}/api/accounts/users/confirm_activation/${token}/`, {});
    if (act.status >= 400 || act.json?.status !== 'success') {
      throw new Error(`激活失败 ${act.status}: ${act.text.slice(0, 300)}`);
    }
    console.log('[ensure-nonadmin] activated', EMAIL);
  }

  const superFlag = sql(`SELECT is_superuser FROM accounts_user WHERE id='${userId}'`);
  if (String(superFlag) === '1') {
    throw new Error(`账号 ${EMAIL} 是超管，不能用于非超管 E2E`);
  }
  console.log('[ensure-nonadmin] ready', { email: EMAIL, userId, is_superuser: superFlag });
  return { email: EMAIL, password: PASSWORD, userId };
}

const isMain =
  Boolean(process.argv[1]) && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url);
if (isMain) {
  ensureE2eNonAdminAccount()
    .then(() => process.exit(0))
    .catch((err) => {
      console.error('[ensure-nonadmin] FAIL', err?.message || err);
      process.exit(1);
    });
}
