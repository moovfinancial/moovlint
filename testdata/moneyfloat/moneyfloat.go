package moneyfloat

type WireTransferRequest struct {
	Amount    float64 // want "Amount uses a float for a monetary value"
	Balance   float32 // want "Balance uses a float for a monetary value"
	Subtotal  float64 // want "Subtotal uses a float for a monetary value"
	Currency  string
	AmountOK  int64
	CentsOK   int64
	Duration  float64
}

type Usd float64

type NamedMoney struct {
	TotalAmount Usd // want "TotalAmount uses a float for a monetary value"
}

func ChargeFee(fee float64) error { // want "fee uses a float for a monetary value"
	return nil
}

func SumTotals(total float64) float64 { // want "total uses a float for a monetary value"
	return total
}

func OKAmountMinor(amount int64) int64 { return amount }

func OKNonMoney(totalSeconds float64) float64 { return totalSeconds }
