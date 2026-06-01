# farm2fork-blockchain

Farm2Fork local Hyperledger Fabric environment.

## Versions

- Hyperledger Fabric: 3.1.4
- Go support line: 1.22.x
- Default pinned Go toolchain: 1.22.12 (see `.env.example`)

## Network defaults

- Channel: `farm2forkchannel`
- Orderer: `orderer.farm2fork.com`
- Peer: `peer0.farm2fork.com`
- History query: `GetHistoryForKey()`

## Intended full flow

The commands below are the target end-to-end flow for this repository once later tasks add the referenced scripts. They are kept here to pin the expected operator workflow without implying that the scripts already exist in this task.

```bash
cp .env.example .env
bash scripts/network-up.sh
bash scripts/create-channel.sh
bash scripts/deploy-chaincode.sh
bash scripts/smoke-test.sh
```
