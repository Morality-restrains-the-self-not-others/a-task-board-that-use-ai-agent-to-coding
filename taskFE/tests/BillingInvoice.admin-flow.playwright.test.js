// @ts-check
/**
 * OPT-20260823-055 / 065：租户申请开票 → 管理员已开具/拒绝/上传专票文件 → 订单详情闭环。
 * 纯 mock，无真实账号。
 */
import { test, expect } from '@playwright/test';
import { clientReachableHost, loadPortConfig } from '../helpers/loadConfYaml.mjs';
import {
  TENANT_ID,
  ORDER_ID,
  INVOICE_APP_ID,
  paidOrderBase,
  installBillingAdminMocks,
  jsonOk,
} from './helpers/billingAdminMocks.js';

const portConfig = loadPortConfig();
const BASE_URL = (
  process.env.PLAYWRIGHT_BILLING_MOCK_ORIGIN ||
  `http://${clientReachableHost(portConfig.vue.host)}:${portConfig.vue.port}`
).replace(/\/$/, '');

const ORDER_PATH = `/tenant/${TENANT_ID}/billing/orders/${ORDER_ID}/`;
const ADMIN_INVOICE_PATH = '/system-admin/order-records/?tab=invoice';
const FILE_URL = `/api/system-admin/invoice-applications/${INVOICE_APP_ID}/invoice-file/`;

function makeState() {
  return {
    appStatus: /** @type {null | 'pending' | 'approved' | 'rejected'} */ (null),
    invoiceType: 'general',
    lastApplyBody: /** @type {object | null} */ (null),
    lastApproveBody: /** @type {object | null} */ (null),
    lastReject: false,
    fileUploaded: false,
    uploads: 0,
  };
}

/** @param {ReturnType<typeof makeState>} state */
function orderPayload(state) {
  const order = paidOrderBase();
  if (state.appStatus) {
    order.invoice_application = { status: state.appStatus };
  }
  if (state.appStatus === 'approved') {
    order.invoices = [{
      id: 'inv-blue-1',
      kind: 'blue',
      status: 'issued',
      amount_yuan: '10.00',
      invoice_type: state.invoiceType,
      invoice_file_url: state.fileUploaded ? FILE_URL : '',
      purpose: 'issue',
    }];
  }
  return order;
}

/** @param {ReturnType<typeof makeState>} state */
function adminRows(state) {
  if (state.appStatus !== 'pending') return [];
  return [{
    id: INVOICE_APP_ID,
    tenant_id: TENANT_ID,
    order_id: ORDER_ID,
    buyer_name: 'E2E 开票抬头',
    buyer_type: 'ORGANIZATION',
    invoice_type: state.invoiceType,
    taxpayer_id: '91110000MA0000000X',
    address: '北京市海淀区测试路 1 号',
    telephone: '010-88888888',
    bank_name: '测试银行总行',
    bank_account: '6222000000000001',
    status: 'pending',
    invoice_file_path: state.fileUploaded ? 'invoices/e2e.pdf' : '',
    invoice_file_url: state.fileUploaded ? FILE_URL : '',
  }];
}

test('租户申请开票后管理员已开具，订单详情显示已开具电子发票', async ({ page }) => {
  test.setTimeout(120000);
  const state = makeState();
  await installBillingAdminMocks(page, BASE_URL, async (route, request) => {
    const url = request.url();
    const method = request.method();
    if (url.includes(`/billing/orders/${ORDER_ID}/invoice-applications/`) && method === 'POST') {
      state.lastApplyBody = JSON.parse(request.postData() || '{}');
      state.appStatus = 'pending';
      state.invoiceType = state.lastApplyBody.invoice_type || 'general';
      await jsonOk(route, { status: 'ok', id: INVOICE_APP_ID });
      return true;
    }
    if (url.includes(`/billing/orders/${ORDER_ID}/`) && method === 'GET') {
      await jsonOk(route, orderPayload(state));
      return true;
    }
    if (url.includes('/api/system-admin/invoice-applications/') && url.includes('/approve/') && method === 'POST') {
      state.lastApproveBody = JSON.parse(request.postData() || '{}');
      state.appStatus = 'approved';
      await jsonOk(route, { status: 'ok' });
      return true;
    }
    if (url.includes('/api/system-admin/invoice-applications/') && method === 'GET') {
      await jsonOk(route, { results: adminRows(state) });
      return true;
    }
    return false;
  });

  await page.goto(`${BASE_URL}${ORDER_PATH}`);
  await page.getByTestId('order-invoice-apply-btn').click();
  await page.getByPlaceholder('姓名或公司全称').fill('E2E 开票抬头');
  await page.getByTestId('order-invoice-apply-confirm').click();
  await expect(page.getByTestId('order-invoice-pending-label')).toBeVisible({ timeout: 15000 });
  expect(state.lastApplyBody && state.lastApplyBody.invoice_type).toBe('general');

  await page.goto(`${BASE_URL}${ADMIN_INVOICE_PATH}`);
  await expect(page.getByTestId('invoice-issued-btn')).toBeVisible({ timeout: 20000 });
  await page.getByTestId('invoice-issued-btn').click();
  await expect(page.getByTestId('invoice-issued-btn')).toHaveCount(0);

  await page.goto(`${BASE_URL}${ORDER_PATH}`);
  await expect(page.getByText('已开具电子发票')).toBeVisible({ timeout: 20000 });
  await expect(page.getByTestId('order-invoice-list')).toContainText('已开具');
  expect(state.lastApproveBody && state.lastApproveBody.note).toBe('已手动开具');
});

test('管理员拒绝后租户可重新申请开票', async ({ page }) => {
  test.setTimeout(120000);
  const state = makeState();
  await installBillingAdminMocks(page, BASE_URL, async (route, request) => {
    const url = request.url();
    const method = request.method();
    if (url.includes(`/billing/orders/${ORDER_ID}/invoice-applications/`) && method === 'POST') {
      state.appStatus = 'pending';
      await jsonOk(route, { status: 'ok', id: INVOICE_APP_ID });
      return true;
    }
    if (url.includes(`/billing/orders/${ORDER_ID}/`) && method === 'GET') {
      await jsonOk(route, orderPayload(state));
      return true;
    }
    if (url.includes('/api/system-admin/invoice-applications/') && url.includes('/reject/') && method === 'POST') {
      state.lastReject = true;
      state.appStatus = 'rejected';
      await jsonOk(route, { status: 'ok' });
      return true;
    }
    if (url.includes('/api/system-admin/invoice-applications/') && method === 'GET') {
      await jsonOk(route, { results: adminRows(state) });
      return true;
    }
    return false;
  });

  await page.goto(`${BASE_URL}${ORDER_PATH}`);
  await page.getByTestId('order-invoice-apply-btn').click();
  await page.getByPlaceholder('姓名或公司全称').fill('E2E 开票抬头');
  await page.getByTestId('order-invoice-apply-confirm').click();
  await expect(page.getByTestId('order-invoice-pending-label')).toBeVisible({ timeout: 15000 });

  await page.goto(`${BASE_URL}${ADMIN_INVOICE_PATH}`);
  await page.getByTestId('invoice-reject-btn').click();
  await expect(page.getByTestId('invoice-reject-btn')).toHaveCount(0);
  expect(state.lastReject).toBe(true);

  await page.goto(`${BASE_URL}${ORDER_PATH}`);
  await expect(page.getByTestId('order-invoice-apply-btn')).toBeVisible({ timeout: 20000 });
  await expect(page.getByText('已开具电子发票')).toHaveCount(0);
});

test('专票申请 + 上传发票文件 + 已开具后租户可查看文件', async ({ page }) => {
  test.setTimeout(120000);
  const state = makeState();
  await installBillingAdminMocks(page, BASE_URL, async (route, request) => {
    const url = request.url();
    const method = request.method();
    if (url.includes(`/billing/orders/${ORDER_ID}/invoice-applications/`) && method === 'POST') {
      state.lastApplyBody = JSON.parse(request.postData() || '{}');
      state.invoiceType = 'special';
      state.appStatus = 'pending';
      await jsonOk(route, { status: 'ok', id: INVOICE_APP_ID });
      return true;
    }
    if (url.includes(`/billing/orders/${ORDER_ID}/`) && method === 'GET') {
      await jsonOk(route, orderPayload(state));
      return true;
    }
    if (url.includes('/invoice-file/') && method === 'POST') {
      state.uploads += 1;
      state.fileUploaded = true;
      await jsonOk(route, { status: 'ok', invoice_file_url: FILE_URL });
      return true;
    }
    if (url.includes('/api/system-admin/invoice-applications/') && url.includes('/approve/') && method === 'POST') {
      state.appStatus = 'approved';
      await jsonOk(route, { status: 'ok' });
      return true;
    }
    if (url.includes('/api/system-admin/invoice-applications/') && method === 'GET') {
      await jsonOk(route, { results: adminRows(state) });
      return true;
    }
    return false;
  });

  await page.goto(`${BASE_URL}${ORDER_PATH}`);
  await page.getByTestId('order-invoice-apply-btn').click();
  await page.getByTestId('order-invoice-type-select').selectOption('special');
  await page.getByPlaceholder('姓名或公司全称').fill('E2E 专票公司');
  await page.getByPlaceholder('统一社会信用代码').fill('91110000MA0000000X');
  await page.getByTestId('order-invoice-address').fill('北京市海淀区测试路 1 号');
  await page.getByTestId('order-invoice-telephone').fill('010-88888888');
  await page.getByTestId('order-invoice-bank-name').fill('测试银行总行');
  await page.getByTestId('order-invoice-bank-account').fill('6222000000000001');
  await page.getByTestId('order-invoice-apply-confirm').click();
  await expect(page.getByTestId('order-invoice-pending-label')).toBeVisible({ timeout: 15000 });
  expect(state.lastApplyBody).toMatchObject({
    invoice_type: 'special',
    type: 'ORGANIZATION',
    taxpayer_id: '91110000MA0000000X',
    address: '北京市海淀区测试路 1 号',
    bank_account: '6222000000000001',
  });

  await page.goto(`${BASE_URL}${ADMIN_INVOICE_PATH}`);
  await expect(page.getByTestId('invoice-type-badge')).toHaveText('专票');
  await expect(page.getByTestId('invoice-special-info')).toContainText('91110000MA0000000X');
  await page.getByTestId('invoice-upload-btn').click();
  await page.getByTestId('invoice-file-input').setInputFiles({
    name: 'fapiao.pdf',
    mimeType: 'application/pdf',
    buffer: Buffer.from('%PDF-1.4 e2e'),
  });
  await expect(page.getByTestId('invoice-file-link')).toBeVisible({ timeout: 15000 });
  expect(state.uploads).toBe(1);
  await page.getByTestId('invoice-issued-btn').click();

  await page.goto(`${BASE_URL}${ORDER_PATH}`);
  await expect(page.getByText('已开具电子发票')).toBeVisible({ timeout: 20000 });
  const fileLink = page.getByTestId('order-invoice-file-link');
  await expect(fileLink).toBeVisible();
  await expect(fileLink).toHaveText('查看发票文件');
  await expect(fileLink).toHaveAttribute('href', FILE_URL);
});
