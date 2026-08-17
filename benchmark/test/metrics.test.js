const assert = require('node:assert/strict');
const test = require('node:test');

const { summarize } = require('../lib/metrics');

test('summarize reports nearest-rank percentiles and wall-clock throughput', () => {
  assert.deepEqual(summarize([40, 10, 30, 20], 1_000, 2_000), {
    count: 4,
    minMs: 10,
    p50Ms: 20,
    p95Ms: 40,
    maxMs: 40,
    elapsedMs: 1_000,
    throughputTps: 4,
  });
});

test('summarize rejects an empty measurement set', () => {
  assert.throws(() => summarize([], 0, 1), /at least one/);
});
