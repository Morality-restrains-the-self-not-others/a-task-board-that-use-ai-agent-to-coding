// @ts-check
import { chromium } from '@playwright/test';
import { loginViaGatewayApi } from '../tests/helpers/gatewayLoginE2e.js';

const SITE = 'http://183.250.1.132:4000';
const GATEWAY = 'http://183.250.1.132:18081';
const TENANT = '850256677331562496';
const WS = '861623708318031872';
const TASK = 'task_12590983282794675865';

const browser = await chromium.launch();
const page = await browser.newPage();
const loginPassword = process.env.PLAYWRIGHT_TEST_PASSWORD;
if (!loginPassword) {
  throw new Error('PLAYWRIGHT_TEST_PASSWORD is required; do not hardcode secrets');
}
const login = await loginViaGatewayApi(page, {
  email: process.env.PLAYWRIGHT_TEST_EMAIL || 'contact@daydaymoney.com',
  password: loginPassword,
  siteOrigin: SITE,
  gatewayOrigin: GATEWAY,
});

const imagesResp = await page.request.get(`${GATEWAY}/api/tenant/${TENANT}/installed-images/`, {
  headers: { Authorization: `Token ${login.token}`, Origin: SITE },
});
console.log('INSTALLED_IMAGES', imagesResp.status(), (await imagesResp.text()).slice(0, 1200));

const payload = {
  task_id: TASK,
  container_image_id: '862588024964280320',
  hardware_config: {
    cpu_cores: '2',
    memory_gb: '4',
    storage_gb: '40',
    instance_type: 'ecs.hfg7.large',
    spot_strategy: 'SpotAsPriceGo',
    spot_duration: 0,
    instance_charge_type: 'PostPaid',
    system_disk_category: 'cloud_essd',
    internet_max_bandwidth_out: 5,
  },
  region_id: 'cn-hongkong',
  zone_id: 'cn-hongkong-b',
  vpc_id: null,
  vswitch_id: null,
  auto_create_vswitch: true,
  cloud_platform_id: '1',
  authorization_id: '862031128628060160',
  filter_options: {
    cores: 2,
    memory: 4,
    io_optimized: true,
    system_disk_category: 'cloud_essd',
    data_disk_category: 'cloud_essd',
    spot_strategy: 'SpotAsPriceGo',
    spot_duration: 0,
    instance_charge_type: 'PostPaid',
    network_category: 'vpc',
  },
  selected_instance: 'ecs.hfg7.large',
  security_group_id: null,
  auto_create_security_group: true,
  bandwidth: 5,
  bandwidth_charging_mode: 'PayByTraffic',
  auto_release_enabled: true,
  auto_release_minutes: 30,
};

const startUrl = `${GATEWAY}/api/tenant/${TENANT}/workspace/${WS}/cloud/compute/start-vm-auto/`;
const resp = await page.request.post(startUrl, {
  headers: { Authorization: `Token ${login.token}`, 'Content-Type': 'application/json', Origin: SITE },
  data: payload,
});
const text = await resp.text();
console.log('START_STATUS', resp.status());
console.log('START_BODY', text.slice(0, 1000));

let eventId = '';
try {
  eventId = JSON.parse(text)?.event_id || '';
} catch {
  /* ignore */
}

for (let i = 0; i < 6; i++) {
  await page.waitForTimeout(10_000);
  const startupUrl = `${GATEWAY}/api/tenant/${TENANT}/workspace/${WS}/cloud/compute/server-startup-status/?task_id=${TASK}${eventId ? `&event_id=${eventId}` : ''}`;
  const st = await page.request.get(startupUrl, {
    headers: { Authorization: `Token ${login.token}`, Origin: SITE },
  });
  const runtime = await page.request.get(`${GATEWAY}/api/tenant/${TENANT}/workspace/${WS}/task/${TASK}/cloud/compute/server-runtime-status/`, {
    headers: { Authorization: `Token ${login.token}`, Origin: SITE },
  });
  console.log(`POLL_${i}`, 'startup=', await st.text(), 'runtime=', await runtime.text());
}

await browser.close();
