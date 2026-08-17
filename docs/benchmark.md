# Fabric benchmark

This benchmark measures the chaincode actually used by Farm2Fork. It does not
add generic `CreateAsset` or `TransferAsset` APIs merely to make the paper
table look conventional.

| Paper label | Actual transaction | Workload data |
| --- | --- | --- |
| CreateAsset | `RecordSupplyChainEvent` with `listed` | One immutable product provenance event per operation. |
| TransferAsset | `RecordSupplyChainEvent` with `shipment_in_transit` | One immutable shipment provenance event per operation, for the corresponding product. |
| QueryHistory | `GetTransactionsByProductId` | The indexed immutable provenance events for one product. |

`GetHistoryForKey` is not the product history query: every outbox message has a
unique immutable ledger key, so its Fabric key-version history is normally
empty. The indexed product query is the application's real provenance history.

## Run

Start and deploy the local network first. The benchmark never resets the
network, so it is safe to run against an already deployed local ledger.

```bash
cd farm2fork-blockchain
bash scripts/network-up.sh
bash scripts/create-channel.sh
bash scripts/deploy-chaincode.sh
node scripts/benchmark.js --operations 100 --concurrency 1,5,10,20
```

Every run generates a UTC-scoped `benchmark-...` ID and uses only IDs under
that prefix. The JSON result is written to `benchmark/results/`; preserve the
specific result file used in the paper under `research/results/` before
committing it. Use `--output-directory <path>` to choose a different location.

## What is measured

For write workloads, one latency sample begins immediately before `peer
chaincode invoke` and ends after that command returns successfully with
`--waitForEvent`; it therefore covers CLI process startup, proposal/endorsement,
ordering, validation/commit event wait, and CLI return. For `QueryHistory`, a
sample begins before `peer chaincode query` and ends after its successful
evaluated-query return. Queries do not commit transactions, so their timing is
reported as query latency, not commit latency.

Each workload runs a fixed number of operations with unthrottled bounded
concurrency. The script records concurrency rather than claiming a fixed
offered TPS. Throughput is successful operations divided by the workload's
wall-clock elapsed time. JSON rows include count, min, p50, p95, max, elapsed
time, and throughput. Percentiles use nearest-rank selection. A non-zero CLI
exit fails the run; do not use a partial or failed run in the paper.

The result also records timestamp, channel, chaincode, peer container, topology,
parameters, and these timing definitions. It is a single-org local-Fabric
benchmark, not a multi-organisation production benchmark.
