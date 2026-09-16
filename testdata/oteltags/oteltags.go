package oteltags

import "time"

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

// SkipMarker fields are ignored by the telemetry library, like untagged ones.
type SkipMarker struct {
	Secret string `otel:"-"`
}

// TimeField serializes to one RFC3339 string attribute.
type TimeField struct {
	CreatedOn time.Time `otel:"created_on"`
}

// amount implements AttributeString, so it records one scalar attribute.
type amount struct{ cents int64 }

func (a amount) AttributeString() string { return "unused" }

type StringerField struct {
	Amount amount `otel:"amount"`
}
