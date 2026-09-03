package enumcast

type FailureReason string

const (
	FailureNone          FailureReason = "none"
	FailureInsufficient  FailureReason = "insufficient_funds"
	FailureCurrency      FailureReason = "currency_unsupported"
)

func badCast(raw string) FailureReason {
	return FailureReason(raw) // want "unchecked conversion to enum type FailureReason"
}

func goodKnownConst() FailureReason {
	return FailureReason("none")
}

func goodSwitch(raw string) FailureReason {
	switch raw {
	case "none":
		return FailureNone
	case "insufficient_funds":
		return FailureInsufficient
	}
	return FailureNone
}

func goodValidation(raw string) (FailureReason, bool) {
	return FailureReason(raw), true
}

func goodNonEnum(raw string) string {
	return raw
}

type PlainString string

func goodNonEnumCast(raw string) PlainString {
	return PlainString(raw)
}
