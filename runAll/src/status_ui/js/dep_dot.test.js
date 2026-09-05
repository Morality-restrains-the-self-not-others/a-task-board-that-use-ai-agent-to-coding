'use strict';

const { describe, it } = require('node:test');
const assert = require('node:assert/strict');
const {
  indexServicesByName,
  resolveDepDotClass
} = require('./dep_dot.js');

describe('dependency indicator dots', () => {
  const classify = function (svc) {
    if (!svc) return 'gray';
    if (svc.status === 'healthy' && svc.readiness && svc.readiness !== 'ready') {
      return 'yellow';
    }
    const map = {
      healthy: 'green', starting: 'yellow', retrying: 'yellow', restarting: 'yellow',
      pending: 'gray', stopped: 'gray', failed: 'red', skipped: 'dark'
    };
    return map[svc.status] || 'gray';
  };

  it('lights green from the live service row when depends_on.status is empty', () => {
    const services = [
      { name: 'docker-redis', status: 'healthy' },
      {
        name: 'ai-monitor',
        status: 'healthy',
        depends_on: [{ name: 'docker-redis', status: '' }]
      }
    ];
    const byName = indexServicesByName(services);
    const dep = services[1].depends_on[0];
    assert.equal(resolveDepDotClass(dep, byName.get(dep.name), classify), 'green');
  });

  it('shows yellow when the live dependency is healthy but readiness is degraded', () => {
    const live = { name: 'saas-backend', status: 'healthy', readiness: 'degraded' };
    const dep = { name: 'saas-backend', status: 'healthy' };
    assert.equal(resolveDepDotClass(dep, live, classify), 'yellow');
  });

  it('falls back to depends_on.status when the live row is missing', () => {
    assert.equal(resolveDepDotClass({ name: 'gone', status: 'failed' }, undefined, classify), 'red');
  });
});
