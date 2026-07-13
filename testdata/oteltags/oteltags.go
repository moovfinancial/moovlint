package oteltags

type GoodModel struct {
	AccountID string `otel:"account_id"`
	PayoutID  string `otel:"payout_id"`
}

type BadCase struct {
	AccountID string `otel:"accountID"` // want "otel tag \"accountID\" must use lower snake case"
}

type BadOmitempty struct {
	AccountID string `otel:"account_id,omitempty"` // want "otel tag must not include omitempty"
}

type BadMap struct {
	Metadata map[string]string `otel:"metadata"` // want "otel tag on field with map"
}
