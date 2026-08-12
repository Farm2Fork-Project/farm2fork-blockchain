package contract

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/model"
)

func validPaymentPayload() model.PaymentPayload {
	return model.PaymentPayload{
		OrderID:  "order-001",
		BuyerID:  "buyer-001",
		FarmerID: "farmer-001",
		Amount:   1500,
		Currency: "PKR",
		Gateway:  "stripe",
		PaidAt:   "2026-06-01T12:00:00Z",
	}
}

func TestValidatePaymentInputNormalizesValidValues(t *testing.T) {
	payment := validPaymentPayload()
	payment.OrderID = " order-001 "
	payment.BuyerID = " buyer-001 "
	payment.FarmerID = " farmer-001 "
	payment.Currency = " PKR "
	payment.Gateway = " stripe "
	payment.PaidAt = " 2026-06-01T12:00:00Z "

	ledgerKey, referenceID, normalized, err := validatePaymentInput(" outbox-payment-001 ", " payment-001 ", payment)

	require.NoError(t, err)
	require.Equal(t, "outbox-payment-001", ledgerKey)
	require.Equal(t, "payment-001", referenceID)
	require.Equal(t, "order-001", normalized.OrderID)
	require.Equal(t, "buyer-001", normalized.BuyerID)
	require.Equal(t, "farmer-001", normalized.FarmerID)
	require.Equal(t, 1500.0, normalized.Amount)
	require.Equal(t, "PKR", normalized.Currency)
	require.Equal(t, "stripe", normalized.Gateway)
	require.Equal(t, "2026-06-01T12:00:00Z", normalized.PaidAt)
}

func TestValidatePaymentInputRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name        string
		ledgerKey   string
		referenceID string
		mutate      func(*model.PaymentPayload)
		expected    string
	}{
		{name: "blank ledger key", ledgerKey: " ", referenceID: "payment-001", expected: "ledger key"},
		{name: "blank reference ID", ledgerKey: "outbox-payment-001", referenceID: " ", expected: "reference ID"},
		{name: "blank order ID", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.OrderID = " " }, expected: "order ID"},
		{name: "blank buyer ID", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.BuyerID = " " }, expected: "buyer ID"},
		{name: "blank farmer ID", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.FarmerID = " " }, expected: "farmer ID"},
		{name: "zero amount", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.Amount = 0 }, expected: "amount"},
		{name: "negative amount", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.Amount = -1 }, expected: "amount"},
		{name: "NaN amount", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.Amount = math.NaN() }, expected: "amount"},
		{name: "infinite amount", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.Amount = math.Inf(1) }, expected: "amount"},
		{name: "invalid currency", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.Currency = "pkr" }, expected: "currency"},
		{name: "unsupported gateway", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.Gateway = "cash" }, expected: "gateway"},
		{name: "malformed paid at", ledgerKey: "outbox-payment-001", referenceID: "payment-001", mutate: func(payment *model.PaymentPayload) { payment.PaidAt = "2026-06-01" }, expected: "paidAt"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payment := validPaymentPayload()
			if test.mutate != nil {
				test.mutate(&payment)
			}

			_, _, _, err := validatePaymentInput(test.ledgerKey, test.referenceID, payment)

			require.ErrorContains(t, err, test.expected)
		})
	}
}

func validSupplyChainPayload() model.SupplyChainPayload {
	return model.SupplyChainPayload{
		ProductID: "product-001",
		FarmerID:  "farmer-001",
		EventType: "listed",
		Location:  "Lahore",
		ActorID:   "farmer-001",
		ActorRole: "farmer",
		Timestamp: "2026-06-01T12:05:00Z",
	}
}

func TestValidateSupplyChainInputNormalizesAllowedEvents(t *testing.T) {
	tests := []struct {
		name           string
		referenceModel string
		eventType      string
		actorRole      string
	}{
		{name: "product listed by farmer", referenceModel: "Product", eventType: "listed", actorRole: "farmer"},
		{name: "shipment assigned to transporter", referenceModel: "Shipment", eventType: "shipment_assigned", actorRole: "transporter"},
		{name: "shipment picked up by transporter", referenceModel: "Shipment", eventType: "shipment_picked_up", actorRole: "transporter"},
		{name: "shipment in transit by transporter", referenceModel: "Shipment", eventType: "shipment_in_transit", actorRole: "transporter"},
		{name: "shipment delivered by transporter", referenceModel: "Shipment", eventType: "shipment_delivered", actorRole: "transporter"},
		{name: "shipment failed by transporter", referenceModel: "Shipment", eventType: "shipment_failed", actorRole: "transporter"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			supplyChain := validSupplyChainPayload()
			supplyChain.EventType = test.eventType
			supplyChain.ActorRole = test.actorRole
			supplyChain.Location = " Lahore "

			ledgerKey, referenceID, referenceModel, normalized, err := validateSupplyChainInput(
				" outbox-event-001 ",
				" reference-001 ",
				" "+test.referenceModel+" ",
				supplyChain,
			)

			require.NoError(t, err)
			require.Equal(t, "outbox-event-001", ledgerKey)
			require.Equal(t, "reference-001", referenceID)
			require.Equal(t, test.referenceModel, referenceModel)
			require.Equal(t, "Lahore", normalized.Location)
			require.Equal(t, test.eventType, normalized.EventType)
			require.Equal(t, test.actorRole, normalized.ActorRole)
		})
	}
}

func TestValidateSupplyChainInputRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name           string
		ledgerKey      string
		referenceID    string
		referenceModel string
		mutate         func(*model.SupplyChainPayload)
		expected       string
	}{
		{name: "blank ledger key", ledgerKey: " ", referenceID: "reference-001", referenceModel: "Product", expected: "ledger key"},
		{name: "blank reference ID", ledgerKey: "outbox-event-001", referenceID: " ", referenceModel: "Product", expected: "reference ID"},
		{name: "unknown reference model", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Order", expected: "reference model"},
		{name: "blank product ID", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.ProductID = " " }, expected: "product ID"},
		{name: "blank farmer ID", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.FarmerID = " " }, expected: "farmer ID"},
		{name: "blank event type", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.EventType = " " }, expected: "event type"},
		{name: "blank location", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.Location = " " }, expected: "location"},
		{name: "blank actor ID", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.ActorID = " " }, expected: "actor ID"},
		{name: "blank actor role", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.ActorRole = " " }, expected: "actor role"},
		{name: "malformed timestamp", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.Timestamp = "2026-06-01" }, expected: "timestamp"},
		{name: "product event must be listed", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.EventType = "shipment_delivered" }, expected: "reference model"},
		{name: "product event must use farmer role", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Product", mutate: func(payload *model.SupplyChainPayload) { payload.ActorRole = "transporter" }, expected: "reference model"},
		{name: "shipment event must use transporter role", ledgerKey: "outbox-event-001", referenceID: "reference-001", referenceModel: "Shipment", mutate: func(payload *model.SupplyChainPayload) {
			payload.EventType = "shipment_delivered"
			payload.ActorRole = "farmer"
		}, expected: "reference model"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			supplyChain := validSupplyChainPayload()
			if test.mutate != nil {
				test.mutate(&supplyChain)
			}

			_, _, _, _, err := validateSupplyChainInput(test.ledgerKey, test.referenceID, test.referenceModel, supplyChain)

			require.ErrorContains(t, err, test.expected)
		})
	}
}
