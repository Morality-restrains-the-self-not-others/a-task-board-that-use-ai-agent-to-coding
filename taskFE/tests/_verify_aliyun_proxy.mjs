import { generateDjangoPasswordHash } from '../app/src/components/PasswordHasher.js';

const GATEWAY = 'http://183.250.1.132:18081';
const SITE = 'http://183.250.1.132:4000';
const TENANT = '850256677331562496';
const AUTH_ID = '862031128628060160';
const email = 'contact@daydaymoney.com';
const password = process.env.PLAYWRIGHT_TEST_PASSWORD;

const originHeader = { Origin: SITE, Accept: 'application/json' };
const privacy = await (
  await fetch(`${GATEWAY}/api/privacy-policy/public/current/`, { headers: originHeader })
).json();
const license = await (
  await fetch(`${GATEWAY}/api/license-agreement/public/current/`, { headers: originHeader })
).json();
const hash = await generateDjangoPasswordHash(password, email);
const loginRes = await fetch(`${GATEWAY}/api/auth/`, {
  method: 'POST',
  headers: { ...originHeader, 'Content-Type': 'application/json' },
  body: JSON.stringify({
    username: email,
    password: hash,
    rememberMe: true,
    accepted_privacy_policy_id: String(privacy?.id || ''),
    accepted_license_agreement_id: String(license?.id || ''),
  }),
});
const login = await loginRes.json();
const token = login.token;
if (!token) {
  console.error('login failed', login);
  process.exit(1);
}

const paths = [
  `/api/cloud/cloud-platform/tenant_id/${TENANT}${AUTH_ID}/cloud/regions/?platform_type=aliyun`,
  `/api/cloud/cloud-platform/tenant_id/${TENANT}${AUTH_ID}/cloud/zones/?platform_type=aliyun&region_id=cn-qingdao`,
  `/api/cloud/cloud-platform/tenant_id/${TENANT}${AUTH_ID}/cloud/available-instances/?platform_type=aliyun&region_id=cn-qingdao&zone_id=cn-qingdao-c`,
];

let failed = false;
for (const path of paths) {
  const res = await fetch(`${GATEWAY}${path}`, {
    headers: { Authorization: `Token ${token}`, Accept: 'application/json', Origin: SITE },
  });
  const text = await res.text();
  console.log('\n===', path.split('?')[0], res.status);
  console.log(text.slice(0, 500));
  if (res.status >= 500 && /proxyconnect/i.test(text)) {
    failed = true;
  }
}
process.exit(failed ? 1 : 0);
