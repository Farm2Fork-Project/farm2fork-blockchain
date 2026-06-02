package main

import (
	"log"

	contractapi "github.com/hyperledger/fabric-contract-api-go/contractapi"

	"farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract"
)

func main() {
	cc, err := contractapi.NewChaincode(&contract.Farm2ForkContract{Contract: contractapi.Contract{Name: "farm2fork-chaincode"}})
	if err != nil {
		log.Fatal(err)
	}

	if err := cc.Start(); err != nil {
		log.Fatal(err)
	}
}
