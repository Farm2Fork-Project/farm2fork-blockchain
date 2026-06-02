package model

type PaymentPayload struct {
	OrderID  string  `json:"orderId"`
	BuyerID  string  `json:"buyerId"`
	FarmerID string  `json:"farmerId"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
	Gateway  string  `json:"gateway"`
	PaidAt   string  `json:"paidAt"`
}

type SupplyChainPayload struct {
	ProductID string `json:"productId"`
	FarmerID  string `json:"farmerId"`
	EventType string `json:"eventType"`
	Location  string `json:"location"`
	ActorID   string `json:"actorId"`
	ActorRole string `json:"actorRole"`
	Timestamp string `json:"timestamp"`
}

type Payload struct {
	Payment     *PaymentPayload     `json:"payment,omitempty"`
	SupplyChain *SupplyChainPayload `json:"supplyChain,omitempty"`
}

type BlockchainTransaction struct {
	Type           string  `json:"type"`
	ReferenceID    string  `json:"referenceId"`
	ReferenceModel string  `json:"referenceModel"`
	TxHash         string  `json:"txHash"`
	BlockNumber    uint64  `json:"blockNumber"`
	ChannelName    string  `json:"channelName"`
	Payload        Payload `json:"payload"`
	Status         string  `json:"status"`
	RetryCount     int     `json:"retryCount"`
	CreatedAt      string  `json:"createdAt"`
}
