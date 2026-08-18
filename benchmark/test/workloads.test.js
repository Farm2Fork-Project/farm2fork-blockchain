const assert = require('node:assert/strict');
const test = require('node:test');

const {
  buildCreateArgs,
  buildHistoryArgs,
  buildTransferArgs,
} = require('../lib/workloads');

test('create workload builds a valid listed product event', () => {
  const args = buildCreateArgs('bench-run', 2);

  assert.equal(args[0], 'bench-run-create-2');
  assert.equal(args[2], 'Product');
  assert.equal(args[5], 'listed');
  assert.equal(args[8], 'farmer');
  assert.match(args[9], /^2026-01-01T00:00:02\.000Z$/);
});

test('transfer workload builds a valid shipment in-transit event', () => {
  const args = buildTransferArgs('bench-run', 2);

  assert.equal(args[2], 'Shipment');
  assert.equal(args[5], 'shipment_in_transit');
  assert.equal(args[8], 'transporter');
  assert.match(args[3], /^bench-run-product-2$/);
});

test('history workload queries the product provenance index', () => {
  assert.deepEqual(buildHistoryArgs('bench-run', 2), ['bench-run-product-2']);
});
