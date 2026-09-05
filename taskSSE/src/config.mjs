import fs from 'node:fs';
import path from 'node:path';
import { execSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

function envPath(env, key) {
  return String(env?.[key] || '').trim();
}

/** CONF_ROOT may be the conf/ dir (contains base.yaml) or the deploy/repo root. */
function resolveExplicitRoot(p) {
  const cleaned = path.resolve(p);
  if (fs.existsSync(path.join(cleaned, 'base.yaml'))) return path.dirname(cleaned);
  if (fs.existsSync(path.join(cleaned, 'conf', 'base.yaml'))) return cleaned;
  return '';
}

/**
 * Locate the directory that contains conf/ (deploy or repo root).
 * Priority matches shareLib/confload.FindConfigRoot: CONF_ROOT → DEPLOY_ROOT → walk.
 * clone-run cwd/__dirname is envs/current/taskSSE; that tree has conf/base.yaml but
 * secrets live in $DEPLOY_ROOT/conf-local — env roots must win over the walk.
 */
export function findRepoRoot(start, env = process.env) {
  const confRoot = envPath(env, 'CONF_ROOT');
  if (confRoot) {
    const resolved = resolveExplicitRoot(confRoot);
    if (resolved) return resolved;
  }
  const deployRoot = envPath(env, 'DEPLOY_ROOT');
  if (deployRoot) {
    const resolved = resolveExplicitRoot(deployRoot);
    if (resolved) return resolved;
  }
  let dir = path.resolve(start);
  for (let i = 0; i < 16; i += 1) {
    // OPT-20260806-057: django 目录退役，锚点切到 conf/base.yaml
    const candidate = path.join(dir, 'conf', 'base.yaml');
    if (fs.existsSync(candidate)) return dir;
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  throw new Error('conf/base.yaml not found');
}

function readYamlFile(p) {
  if (!fs.existsSync(p)) return {};
  const text = fs.readFileSync(p, 'utf8');
  try {
    return JSON.parse(text);
  } catch {
    const out = {};
    for (const line of text.split('\n')) {
      const m = line.match(/^(\w+):\s*(.+)$/);
      if (m) out[m[1]] = m[2].replace(/(?:^["']|["']$)/g, '');
    }
    return out;
  }
}

function deepMerge(base, overlay) {
  const out = { ...base };
  for (const [k, v] of Object.entries(overlay || {})) {
    if (
      v &&
      typeof v === 'object' &&
      !Array.isArray(v) &&
      base[k] &&
      typeof base[k] === 'object' &&
      !Array.isArray(base[k])
    ) {
      out[k] = deepMerge(base[k], v);
    } else if (v !== undefined) {
      out[k] = v;
    }
  }
  return out;
}

function loadYamlApp(repoRoot, app) {
  const mainPath = path.join(repoRoot, 'conf', app, 'config.yaml');
  const localPath = path.join(repoRoot, 'conf-local', app, 'config.yaml');
  const fragPath = path.join(repoRoot, 'conf', app, 'docker-infra.yaml');
  const fragLocalPath = path.join(repoRoot, 'conf-local', app, 'docker-infra.yaml');
  let cfg = readYamlFile(mainPath);
  cfg = deepMerge(cfg, readYamlFile(localPath));
  cfg = deepMerge(cfg, readYamlFile(fragPath));
  cfg = deepMerge(cfg, readYamlFile(fragLocalPath));
  return cfg;
}

function loadConfRead(repoRoot, jsonKey) {
  try {
    const script = path.join(repoRoot, 'runAll/scripts', 'conf-read.py');
    const raw = execSync(`python3 "${script}" ${jsonKey} --json`, {
      cwd: repoRoot,
      encoding: 'utf8',
      stdio: ['ignore', 'pipe', 'pipe'],
    });
    const block = JSON.parse(raw);
    if (!block || (typeof block === 'object' && Object.keys(block).length === 0)) {
      return null;
    }
    return block;
  } catch {
    return null;
  }
}

/** Merged conf/gateway/task-sse (incl. docker-infra + conf-local) via Python loader. */
function loadTaskSseBlock(repoRoot) {
  const block = loadConfRead(repoRoot, 'taskSSE');
  if (block) {
    return block;
  }
  console.warn('[taskSSE] conf-read.py returned empty for "taskSSE" — falling back to YAML config loader');
  return loadYamlApp(repoRoot, 'gateway/task-sse');
}

export function loadPortConfig() {
  const repoRoot = findRepoRoot(path.join(__dirname, '..', '..'), process.env);
  try {
    const script = path.join(repoRoot, 'runAll/scripts', 'conf-read.py');
    const raw = execSync(`python3 "${script}" snapshot-json`, { cwd: repoRoot, encoding: 'utf8' });
    const data = JSON.parse(raw);
    return { ...data, __repoRoot: repoRoot };
  } catch {
    return {
      // OPT-20260806-057: django 块退役
      taskSSE: loadYamlApp(repoRoot, 'gateway/task-sse'),
      __repoRoot: repoRoot,
    };
  }
}

export function loadTaskSseConfig(env = process.env) {
  const repoRoot = findRepoRoot(path.join(__dirname, '..', '..'), env);
  const block = loadTaskSseBlock(repoRoot);
  // SSOT（约束 29）：gatewayInternalSecret 由本目录 conf/gateway/task-sse/ 声明，
  // 主机密钥经 conf-local/gateway/task-sse/ overlay（ADR-0054），与 APISIX
  // proxy-rewrite 写入的 X-TaskGateway-Internal-Secret 同源。禁止运行时直读 task-gateway。
  const host = String(env.TASK_SSE_HOST || block.host || '127.0.0.1').trim();
  const port = Number(env.TASK_SSE_PORT || block.port || 8798);
  const transport = String(env.TASK_SSE_TRANSPORT || block.transport || 'redis').trim().toLowerCase();
  const secret = String(env.TASK_SSE_SECRET || block.secret || 'dev-secret').trim();
  const gatewayInternalSecret = String(
    env.TASK_GATEWAY_INTERNAL_SECRET ||
      env.TASK_SSE_GATEWAY_INTERNAL_SECRET ||
      block.gatewayInternalSecret ||
      '',
  ).trim();
  const heartbeatSec = Number(env.TASK_SSE_HEARTBEAT_SEC || block.heartbeatIntervalSec || 5);
  const maxConnections = Number(env.TASK_SSE_MAX_CONNECTIONS || block.maxConnections || 500);
  const redis = {
    host: String(env.TASK_SSE_REDIS_HOST || block.redis?.host || '127.0.0.1').trim(),
    port: Number(env.TASK_SSE_REDIS_PORT || block.redis?.port || 6379),
    channelPrefix: String(env.TASK_SSE_REDIS_PREFIX || block.redis?.channelPrefix || 'sse:').trim(),
  };
  const kafka = {
    bootstrapServers: String(
      env.TASK_SSE_KAFKA_BOOTSTRAP || block.kafka?.bootstrapServers || 'localhost:9093',
    ).trim(),
    topic: String(env.TASK_SSE_KAFKA_TOPIC || block.kafka?.topic || 'sse-message').trim(),
    groupId: String(env.TASK_SSE_KAFKA_GROUP_ID || block.kafka?.groupId || 'task-sse-consumer').trim(),
  };
  return {
    host,
    port,
    transport,
    secret,
    gatewayInternalSecret,
    heartbeatSec,
    maxConnections: Number.isFinite(maxConnections) ? maxConnections : 500,
    redis,
    kafka,
    enabled: block.enabled !== false,
  };
}
