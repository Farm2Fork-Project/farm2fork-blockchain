# farm2fork-blockchain

Farm2Fork local Hyperledger Fabric environment.

## Versions

- Hyperledger Fabric: 3.1.4
- Go support line: 1.23.x
- Default pinned Go toolchain: 1.23.0 (see `.env.example`)

## Network defaults

- Channel: `farm2forkchannel`
- Orderer: `orderer.farm2fork.com`
- Peer: `peer0.farm2fork.com`
- History query: `GetHistoryForKey()`

## Full flow

```bash
cp .env.example .env
bash scripts/network-up.sh
bash scripts/create-channel.sh
bash scripts/deploy-chaincode.sh
bash scripts/smoke-test.sh
bash scripts/network-down.sh
```

`scripts/smoke-test.sh` resets the local Fabric network by default so it always tests
the chaincode currently on disk. Set `SMOKE_RESET_NETWORK=false` if you need to
preserve the current local ledger while running the smoke test.

## Limitations

- Single-org only
- No CouchDB
- History uses `GetHistoryForKey()`
- No backend SDK integration yet
