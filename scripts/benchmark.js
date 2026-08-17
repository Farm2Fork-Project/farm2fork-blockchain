#!/usr/bin/env node

const { execFile } = require('node:child_process');
const { mkdir, writeFile } = require('node:fs/promises');
const path = require('node:path');
const { promisify } = require('node:util');
const { performance } = require('node:perf_hooks');
const { summarize } = require('../benchmark/lib/metrics');
const {
  buildCreateArgs,
  buildHistoryArgs,
  buildTransferArgs,
} = require('../benchmark/lib/workloads');

const execFileAsync = promisify(execFile);
const CHANNEL_NAME = process.env.CHANNEL_NAME ?? 'farm2forkchannel';
const CHAINCODE_NAME = process.env.CHAINCODE_NAME ?? 'farm2fork-chaincode';
const PEER_CONTAINER_NAME = process.env.PEER_CONTAINER_NAME ?? 'peer0.farm2fork.com';
const ORDERER_ADDRESS = process.env.ORDERER_ADDRESS ?? 'orderer.farm2fork.com:7050';
const ORDERER_CA_FILE = '/etc/hyperledger/fabric/orderer/tls/ca.crt';
const DEFAULT_OPTIONS = {
  operations: 100,
  concurrency: [1, 5, 10, 20],
  outputDirectory: 'benchmark/results',
};

function parseArgs(argv) {
  const options = { ...DEFAULT_OPTIONS };
  for (let index = 0; index < argv.length; index += 2) {
    const flag = argv[index];
    const value = argv[index + 1];
    if (!value || !['--operations', '--concurrency', '--output-directory'].includes(flag)) {
      throw new Error(`Unknown or incomplete argument: ${flag ?? ''}`.trim());
    }
    if (flag === '--operations') {
      options.operations = Number(value);
    } else if (flag === '--concurrency') {
      options.concurrency = value.split(',').map(Number);
    } else {
      options.outputDirectory = value;
    }
  }
  if (!Number.isInteger(options.operations) || options.operations < 1) {
    throw new Error('--operations must be a positive integer');
  }
  if (
    options.concurrency.length === 0 ||
    options.concurrency.some((value) => !Number.isInteger(value) || value < 1)
  ) {
    throw new Error('--concurrency must be a comma-separated list of positive integers');
  }
  return options;
}

async function run() {
  const options = parseArgs(process.argv.slice(2));
  await assertPeerIsRunning();
  const runId = `benchmark-${new Date().toISOString().replace(/[:.]/g, '-')}`;
  const rows = [];

  for (const concurrency of options.concurrency) {
    rows.push(
      await runWorkload({
        runId,
        workload: 'CreateAsset',
        transaction: 'RecordSupplyChainEvent(listed)',
        builder: buildCreateArgs,
        operations: options.operations,
        concurrency,
        invoke: true,
      }),
    );
    rows.push(
      await runWorkload({
        runId,
        workload: 'TransferAsset',
        transaction: 'RecordSupplyChainEvent(shipment_in_transit)',
        builder: buildTransferArgs,
        operations: options.operations,
        concurrency,
        invoke: true,
      }),
    );
    rows.push(
      await runWorkload({
        runId,
        workload: 'QueryHistory',
        transaction: 'GetTransactionsByProductId',
        builder: buildHistoryArgs,
        operations: options.operations,
        concurrency,
        invoke: false,
      }),
    );
  }

  const result = {
    schemaVersion: 1,
    measuredAt: new Date().toISOString(),
    runId,
    fabric: {
      channelName: CHANNEL_NAME,
      chaincodeName: CHAINCODE_NAME,
      peerContainerName: PEER_CONTAINER_NAME,
      topology: 'single organization, one peer, one orderer, LevelDB',
    },
    parameters: options,
    measurement: {
      writeLatency: 'CLI invoke start through successful Fabric --waitForEvent completion',
      queryLatency: 'CLI query start through successful evaluated-query completion',
      throughput: 'successful operations divided by wall-clock duration of each workload',
      offeredRate: 'unthrottled bounded concurrency; concurrency is recorded, not TPS rate-limited',
    },
    rows,
  };
  const outputDirectory = path.resolve(options.outputDirectory);
  await mkdir(outputDirectory, { recursive: true });
  const resultPath = path.join(outputDirectory, `${runId}.json`);
  await writeFile(resultPath, `${JSON.stringify(result, null, 2)}\n`);
  process.stdout.write(`${renderPaperTable(rows)}\n\nResult: ${resultPath}\n`);
}

async function assertPeerIsRunning() {
  const { stdout } = await execFileAsync('docker', ['ps', '--format', '{{.Names}}']);
  if (!stdout.split(/\r?\n/).includes(PEER_CONTAINER_NAME)) {
    throw new Error(
      `Fabric peer container ${PEER_CONTAINER_NAME} is not running. Start and deploy the local network first.`,
    );
  }
}

async function runWorkload({
  runId,
  workload,
  transaction,
  builder,
  operations,
  concurrency,
  invoke,
}) {
  const indices = Array.from({ length: operations }, (_, index) => index + 1);
  const startedAtMs = performance.now();
  const latenciesMs = await runBounded(indices, concurrency, async (index) => {
    const startedAt = performance.now();
    await submit(invoke, builder(runId, index));
    return performance.now() - startedAt;
  });
  const metrics = summarize(latenciesMs, startedAtMs, performance.now());
  return { workload, transaction, concurrency, metrics };
}

async function runBounded(items, concurrency, operation) {
  const results = [];
  let nextIndex = 0;
  const workers = Array.from(
    { length: Math.min(concurrency, items.length) },
    async () => {
      while (nextIndex < items.length) {
        const index = nextIndex;
        nextIndex += 1;
        results.push(await operation(items[index]));
      }
    },
  );
  await Promise.all(workers);
  return results;
}

async function submit(invoke, args) {
  const invocation = JSON.stringify({
    Args: [
      invoke ? 'RecordSupplyChainEvent' : 'GetTransactionsByProductId',
      ...args,
    ],
  });
  const peerArgs = invoke
    ? [
        'peer',
        'chaincode',
        'invoke',
        '-o',
        ORDERER_ADDRESS,
        '--tls',
        '--cafile',
        ORDERER_CA_FILE,
        '--waitForEvent',
        '--waitForEventTimeout',
        '60s',
        '-C',
        CHANNEL_NAME,
        '-n',
        CHAINCODE_NAME,
        '-c',
        invocation,
      ]
    : ['peer', 'chaincode', 'query', '-C', CHANNEL_NAME, '-n', CHAINCODE_NAME, '-c', invocation];
  await execFileAsync('docker', buildDockerPeerArgs(peerArgs), {
    maxBuffer: 10 * 1024 * 1024,
  });
}

function buildDockerPeerArgs(peerArgs) {
  return [
    'exec',
    '-e',
    'CORE_PEER_LOCALMSPID=Farm2ForkMSP',
    '-e',
    'CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/admin/msp',
    '-e',
    'CORE_PEER_TLS_ENABLED=true',
    '-e',
    'CORE_PEER_ADDRESS=peer0.farm2fork.com:7051',
    '-e',
    'CORE_PEER_TLS_ROOTCERT_FILE=/etc/hyperledger/fabric/tls/ca.crt',
    PEER_CONTAINER_NAME,
    ...peerArgs,
  ];
}

function renderPaperTable(rows) {
  const header =
    '| Paper workload | Chaincode transaction | Concurrency | Throughput (ops/s) | Latency p50 (ms) | Latency p95 (ms) |';
  const divider = '| --- | --- | ---: | ---: | ---: | ---: |';
  const body = rows.map(
    ({ workload, transaction, concurrency, metrics }) =>
      `| ${workload} | ${transaction} | ${concurrency} | ${metrics.throughputTps.toFixed(2)} | ${metrics.p50Ms.toFixed(2)} | ${metrics.p95Ms.toFixed(2)} |`,
  );
  return [header, divider, ...body].join('\n');
}

if (require.main === module) {
  run().catch((error) => {
    process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
    process.exitCode = 1;
  });
}

module.exports = { buildDockerPeerArgs, parseArgs, renderPaperTable };
