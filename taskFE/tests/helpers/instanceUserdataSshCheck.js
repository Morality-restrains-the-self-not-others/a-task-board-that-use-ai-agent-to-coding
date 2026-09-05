// @ts-check
/**
 * 使用本机 OpenSSH 客户端（PATH 中需有 `ssh`）连入 ECS，校验 UserData 生成的
 * /root/init_from_task2app.sh.log 存在且非空，并可选校验 sshd drop-in。
 *
 * 在实例刚创建、cloud-init 尚未跑完时会失败，由调用方重试。
 */
import { execFile } from 'child_process';
import { mkdtemp, rm, writeFile } from 'fs/promises';
import { tmpdir } from 'os';
import path from 'path';
import { promisify } from 'util';

const execFileAsync = promisify(execFile);

/** 阿里云 / 常见镜像上可能允许 SSH 的用户名（UserData 将公钥写入 root，但部分镜像禁止 root 密码/密钥登录） */
const SSH_USER_CANDIDATES = ['root', 'ecs-user', 'ubuntu', 'admin'];

/**
 * @param {string} privateKeyPem OpenSSH PEM 私钥全文
 * @param {string} publicIp 公网 IPv4
 * @param {{
 *   maxAttempts?: number,
 *   intervalMs?: number,
 *   requireSshdDropin?: boolean,
 *   sshUsers?: string[],
 *   expectedContainerImageRef?: string
 * }} [opts]
 * @returns {Promise<{ stdout: string, stderr: string, sshUser: string }>}
 */
export async function assertInitFromTask2appLogViaSsh (privateKeyPem, publicIp, opts = {}) {
  const maxAttempts = opts.maxAttempts ?? 32;
  const intervalMs = opts.intervalMs ?? 15000;
  const requireSshdDropin = opts.requireSshdDropin === true;
  const expectedContainerImageRef = String(opts.expectedContainerImageRef || '').trim();
  const sshUsers = Array.isArray(opts.sshUsers) && opts.sshUsers.length ? opts.sshUsers : SSH_USER_CANDIDATES;

  const dir = await mkdtemp(path.join(tmpdir(), 'e2e-probe-ssh-'));
  const keyPath = path.join(dir, 'probe.pem');
  await writeFile(keyPath, privateKeyPem, { mode: 0o600 });

  const remoteScript = `
set -e
run() { if [ "$(id -u)" -eq 0 ]; then "$@"; else sudo -n "$@"; fi; }
run test -f /root/init_from_task2app.sh.log || { echo TASK2APP_E2E_MISSING_LOG; exit 2; }
echo TASK2APP_E2E_LOG_OK
run wc -c /root/init_from_task2app.sh.log
run tail -n 200 /root/init_from_task2app.sh.log
run test -f /root/init_from_task2app.sh || { echo TASK2APP_E2E_MISSING_INIT_SCRIPT; exit 2; }
run sed -n '1,240p' /root/init_from_task2app.sh
if run grep -q '__TASK2APP_CONTAINER_IMAGE__' /root/init_from_task2app.sh; then
  echo TASK2APP_E2E_PLACEHOLDER_STILL_PRESENT
  exit 3
fi
echo TASK2APP_E2E_NO_IMAGE_PLACEHOLDER
if run docker ps --filter name=task2app-container --format '{{.Names}} {{.Status}}' 2>/dev/null | grep -q '^task2app-container'; then
  echo TASK2APP_E2E_CONTAINER_RUNNING
else
  echo TASK2APP_E2E_CONTAINER_NOT_RUNNING
  run docker ps -a --filter name=task2app-container --format '{{.Names}} {{.Status}}' 2>/dev/null || true
fi
run test -f /etc/ssh/sshd_config.d/99-task2app.conf && echo TASK2APP_E2E_SSHD_DROPIN_OK || true
`.trim();

  const sshOpts = [
    '-o', 'BatchMode=yes',
    '-o', 'StrictHostKeyChecking=accept-new',
    '-o', 'UserKnownHostsFile=/dev/null',
    '-o', 'ConnectTimeout=25',
    '-i', keyPath
  ];

  let lastErr;
  try {
    for (let i = 0; i < maxAttempts; i++) {
      for (const sshUser of sshUsers) {
        const sshBase = [...sshOpts, `${sshUser}@${publicIp}`, 'bash', '-lc', remoteScript];
        try {
          const { stdout, stderr } = await execFileAsync('ssh', sshBase, {
            maxBuffer: 4 * 1024 * 1024
          });
          const combined = `${stdout}\n${stderr}`;
          if (combined.includes('TASK2APP_E2E_MISSING_LOG')) {
            throw new Error(`log file missing on instance:\n${combined}`);
          }
          if (combined.includes('TASK2APP_E2E_MISSING_INIT_SCRIPT')) {
            throw new Error(`init_from_task2app.sh missing on instance:\n${combined}`);
          }
          if (!combined.includes('TASK2APP_E2E_LOG_OK')) {
            throw new Error(`marker TASK2APP_E2E_LOG_OK not found:\n${combined}`);
          }
          const wcMatch = stdout.match(/(\d+)\s+\/root\/init_from_task2app\.sh\.log/);
          if (!wcMatch) {
            throw new Error(`unexpected wc output:\n${stdout}`);
          }
          const bytes = Number(wcMatch[1]);
          if (!Number.isFinite(bytes) || bytes < 1) {
            throw new Error(`init_from_task2app.sh.log empty or invalid wc: ${wcMatch[0]}`);
          }
          if (expectedContainerImageRef && !combined.includes(expectedContainerImageRef)) {
            throw new Error(
              `init_from_task2app.sh does not contain expected image ref: ${expectedContainerImageRef}\n${combined}`
            );
          }
          if (combined.includes('TASK2APP_E2E_PLACEHOLDER_STILL_PRESENT')) {
            throw new Error(`init_from_task2app.sh still contains __TASK2APP_CONTAINER_IMAGE__ placeholder:\n${combined}`);
          }
          if (opts.requireContainerRunning && !combined.includes('TASK2APP_E2E_CONTAINER_RUNNING')) {
            throw new Error(`docker container task2app-container not running:\n${combined}`);
          }
          if (requireSshdDropin && !combined.includes('TASK2APP_E2E_SSHD_DROPIN_OK')) {
            throw new Error('expected /etc/ssh/sshd_config.d/99-task2app.conf (set requireSshdDropin only when UserData 含 SSH 模板)');
          }
          return { stdout, stderr, sshUser };
        } catch (e) {
          lastErr = e;
          const msg = e && typeof e === 'object' && 'message' in e ? String(e.message) : String(e);
          if (msg.includes('Permission denied') || msg.includes('publickey')) {
            continue;
          }
          if (msg.includes('TASK2APP_E2E') || msg.includes('unexpected wc') || msg.includes('marker')) {
            throw e;
          }
          if (msg.includes('Connection timed out') || msg.includes('Connection refused') || msg.includes('No route')) {
            break;
          }
          throw e;
        }
      }
      await new Promise((r) => setTimeout(r, intervalMs));
    }
    throw lastErr;
  } finally {
    await rm(dir, { recursive: true, force: true }).catch(() => {});
  }
}
