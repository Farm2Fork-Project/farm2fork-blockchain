package contract

import (
	"encoding/json"
	"errors"

	contractapi "github.com/hyperledger/fabric-contract-api-go/v2/contractapi"

	"farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/model"
)

var errTransactionNotFound = errors.New("transaction not found")

type Farm2ForkContract struct{}

func requireTransactionContext(ctx contractapi.TransactionContextInterface) error {
	if ctx == nil || ctx.GetStub() == nil {
		return errors.New("transaction context is required")
	}

	return nil
}

func buildBaseTransaction(
	ctx contractapi.TransactionContextInterface,
	referenceID string,
	referenceModel string,
	transactionType string,
	createdAt string,
) *model.BlockchainTransaction {
	return &model.BlockchainTransaction{
		Type:           transactionType,
		ReferenceID:    referenceID,
		ReferenceModel: referenceModel,
		TxHash:         ctx.GetStub().GetTxID(),
		BlockNumber:    0,
		ChannelName:    ctx.GetStub().GetChannelID(),
		Status:         "confirmed",
		RetryCount:     0,
		CreatedAt:      createdAt,
	}
}

func persistTransaction(ctx contractapi.TransactionContextInterface, referenceID string, tx *model.BlockchainTransaction) error {
	bytes, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(referenceID, bytes)
}

func loadTransaction(ctx contractapi.TransactionContextInterface, referenceID string) (*model.BlockchainTransaction, error) {
	bytes, err := ctx.GetStub().GetState(referenceID)
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, errTransactionNotFound
	}

	var tx model.BlockchainTransaction
	if err := json.Unmarshal(bytes, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func loadHistory(ctx contractapi.TransactionContextInterface, referenceID string) ([]*model.BlockchainTransaction, error) {
	iter, err := ctx.GetStub().GetHistoryForKey(referenceID)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	history := make([]*model.BlockchainTransaction, 0)
	for iter.HasNext() {
		response, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if len(response.Value) == 0 {
			continue
		}

		var tx model.BlockchainTransaction
		if err := json.Unmarshal(response.Value, &tx); err != nil {
			return nil, err
		}
		history = append(history, &tx)
	}

	return history, nil
}

func (c *Farm2ForkContract) RecordPayment(
	ctx contractapi.TransactionContextInterface,
	referenceID string,
	orderID string,
	buyerID string,
	farmerID string,
	amount float64,
	currency string,
	gateway string,
	paidAt string,
) (*model.BlockchainTransaction, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return nil, err
	}

	tx := buildBaseTransaction(ctx, referenceID, "Payment", "payment", paidAt)
	tx.Payload.Payment = &model.PaymentPayload{
		OrderID:  orderID,
		BuyerID:  buyerID,
		FarmerID: farmerID,
		Amount:   amount,
		Currency: currency,
		Gateway:  gateway,
		PaidAt:   paidAt,
	}

	if err := persistTransaction(ctx, referenceID, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (c *Farm2ForkContract) RecordSupplyChainEvent(
	ctx contractapi.TransactionContextInterface,
	referenceID string,
	referenceModel string,
	productID string,
	farmerID string,
	eventType string,
	location string,
	actorID string,
	actorRole string,
	timestamp string,
) (*model.BlockchainTransaction, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return nil, err
	}

	tx := buildBaseTransaction(ctx, referenceID, referenceModel, "supply_chain_event", timestamp)
	tx.Payload.SupplyChain = &model.SupplyChainPayload{
		ProductID: productID,
		FarmerID:  farmerID,
		EventType: eventType,
		Location:  location,
		ActorID:   actorID,
		ActorRole: actorRole,
		Timestamp: timestamp,
	}

	if err := persistTransaction(ctx, referenceID, tx); err != nil {
		return nil, err
	}

	return tx, nil
}

func (c *Farm2ForkContract) GetTransactionByReferenceId(
	ctx contractapi.TransactionContextInterface,
	referenceID string,
) (*model.BlockchainTransaction, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return nil, err
	}

	return loadTransaction(ctx, referenceID)
}

func (c *Farm2ForkContract) GetHistoryForKey(
	ctx contractapi.TransactionContextInterface,
	referenceID string,
) ([]*model.BlockchainTransaction, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return nil, err
	}

	return loadHistory(ctx, referenceID)
}
