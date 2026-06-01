# Farm2Fork Blockchain Architecture Notes

## Pinned platform constraints

- Hyperledger Fabric version: 3.1.4
- Go support line: 1.22.x
- Default pinned Go version: 1.22.12
- Channel name: farm2forkchannel
- Orderer: orderer.farm2fork.com
- Peer: peer0.farm2fork.com

## Ledger behavior constraints

- History query primitive: GetHistoryForKey()
- Initial network shape: single org, single peer, single orderer
- State database expectation for the first pass: LevelDB-backed history verification, not CouchDB

## Scope notes

- Not in scope: CouchDB, Fabric CA, backend SDK integration, multi-org
- Purpose of these notes: pin the architecture contract that later network and chaincode tasks must implement
