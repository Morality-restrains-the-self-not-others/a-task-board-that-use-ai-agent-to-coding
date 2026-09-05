import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import path from 'node:path';
import { after, describe, it } from 'node:test';

import { findRepoRoot, loadTaskSseConfig } from './config.mjs';

const fixtures = [];

function makeDeployLayout() {
  const deploy = mkdtempSync(path.join(tmpdir(), 'task-sse-root-'));
  fixtures.push(deploy);
  mkdirSync(path.join(deploy, 'conf', 'gateway', 'task-sse'), { recursive: true });
  mkdirSync(path.join(deploy, 'conf', 'gateway', 'task-gateway'), { recursive: true });
  mkdirSync(path.join(deploy, 'conf-local', 'gateway', 'task-gateway'), { recursive: true });
  // OPT-20260902-001：密钥须叠在 task-sse 本目录 conf-local，禁止跨 app 直读 task-gateway。
  mkdirSync(path.join(deploy, 'conf-local', 'gateway', 'task-sse'), { recursive: true });
  mkdirSync(path.join(deploy, 'envs', 'current', 'conf'), { recursive: true });
  mkdirSync(path.join(deploy, 'envs', 'current', 'taskSSE', 'src'), { recursive: true });
  writeFileSync(path.join(deploy, 'conf', 'base.yaml'), 'scheme: https\n');
  writeFileSync(path.join(deploy, 'envs', 'current', 'conf', 'base.yaml'), 'scheme: https\n');
  writeFileSync(
    path.join(deploy, 'conf', 'gateway', 'task-sse', 'config.yaml'),
    'host: 0.0.0.0\nport: 8798\nsecret: ""\ngatewayInternalSecret: ""\n',
  );
  writeFileSync(
    path.join(deploy, 'conf', 'gateway', 'task-gateway', 'config.yaml'),
    'gatewayInternalSecret: ""\n',
  );
  writeFileSync(
    path.join(deploy, 'conf-local', 'gateway', 'task-gateway', 'config.yaml'),
    'gatewayInternalSecret: from-deploy-conf-local\n',
  );
  writeFileSync(
    path.join(deploy, 'conf-local', 'gateway', 'task-sse', 'config.yaml'),
    'gatewayInternalSecret: from-deploy-conf-local\n',
  );
  return deploy;
}

after(() => {
  for (const dir of fixtures) {
    rmSync(dir, { recursive: true, force: true });
  }
});

describe('findRepoRoot clone-run CONF_ROOT', () => {
  it('honors CONF_ROOT pointing at conf/ over nested envs/current conf/base.yaml', () => {
    const deploy = makeDeployLayout();
    const nestedSrc = path.join(deploy, 'envs', 'current', 'taskSSE', 'src');
    const got = findRepoRoot(nestedSrc, {
      CONF_ROOT: path.join(deploy, 'conf'),
      DEPLOY_ROOT: deploy,
    });
    assert.equal(path.resolve(got), path.resolve(deploy));
  });

  it('honors DEPLOY_ROOT when CONF_ROOT is unset', () => {
    const deploy = makeDeployLayout();
    const nestedSrc = path.join(deploy, 'envs', 'current', 'taskSSE', 'src');
    const got = findRepoRoot(nestedSrc, { DEPLOY_ROOT: deploy });
    assert.equal(path.resolve(got), path.resolve(deploy));
  });

  it('walks to first conf/base.yaml when CONF_ROOT and DEPLOY_ROOT are empty', () => {
    const deploy = makeDeployLayout();
    const nestedSrc = path.join(deploy, 'envs', 'current', 'taskSSE', 'src');
    const got = findRepoRoot(nestedSrc, {});
    assert.equal(path.resolve(got), path.resolve(path.join(deploy, 'envs', 'current')));
  });
});

describe('loadTaskSseConfig clone-run overlay', () => {
  it('loads gatewayInternalSecret from deploy conf-local when CONF_ROOT is set', () => {
    const deploy = makeDeployLayout();
    const cfg = loadTaskSseConfig({
      CONF_ROOT: path.join(deploy, 'conf'),
      DEPLOY_ROOT: deploy,
    });
    assert.equal(cfg.gatewayInternalSecret, 'from-deploy-conf-local');
  });
});
