# farm2fork-blockchain

Farm2Fork local Hyperledger Fabric environment.

## Versions

- Hyperledger Fabric: 3.1.4
- Go: 1.22.x

## Network defaults

- Channel: `farm2forkchannel`
- Orderer: `orderer.farm2fork.com`
- Peer: `peer0.farm2fork.com`
- History query: `GetHistoryForKey()`

## Quick start

```bash
cp .env.example .env
bash scripts/network-up.sh
bash scripts/create-channel.sh
bash scripts/deploy-chaincode.sh
bash scripts/smoke-test.sh
```
