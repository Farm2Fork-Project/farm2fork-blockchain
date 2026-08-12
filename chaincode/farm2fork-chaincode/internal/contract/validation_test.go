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
