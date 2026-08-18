const assert = require('node:assert/strict');
const test = require('node:test');

const {
  buildDockerPeerArgs,
  parseArgs,
  renderPaperTable,
} = require('../../scripts/benchmark');

test('parseArgs accepts an explicit bounded concurrency matrix', () => {
  assert.deepEqual(parseArgs(['--operations', '4', '--concurrency', '1,5']), {
    operations: 4,
    concurrency: [1, 5],
    outputDirectory: 'benchmark/results',
  });
});

test('parseArgs rejects invalid operation counts', () => {
  assert.throws(() => parseArgs(['--operations', '0']), /operations/);
});

test('renderPaperTable maps the exact benchmark operations to paper concepts', () => {
  const markdown = renderPaperTable([
    {
      workload: 'CreateAsset',
      transaction: 'RecordSupplyChainEvent(listed)',
      concurrency: 1,
      metrics: { throughputTps: 3.2, p50Ms: 10.1, p95Ms: 12.4 },
    },
  ]);

  assert.match(markdown, /CreateAsset/);
  assert.match(markdown, /RecordSupplyChainEvent\(listed\)/);
  assert.match(markdown, /3\.20/);
});

test('buildDockerPeerArgs uses the local Fabric administrator identity', () => {
  const args = buildDockerPeerArgs(['peer', 'chaincode', 'query']);

  assert.deepEqual(args.slice(0, 7), [
    'exec',
    '-e',
    'CORE_PEER_LOCALMSPID=Farm2ForkMSP',
    '-e',
    'CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/admin/msp',
    '-e',
    'CORE_PEER_TLS_ENABLED=true',
  ]);
  assert.ok(args.includes('CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/admin/msp'));
  assert.ok(args.includes('peer0.farm2fork.com'));
});
