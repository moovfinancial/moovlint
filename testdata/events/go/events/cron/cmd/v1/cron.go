package v1

// Trigger is a Wrath of Cron trigger. Real type lives in events/go/events/cron/cmd/v1.
type Trigger struct {
	Name string
}

// ScheduleTrigger is a Wrath of Cron schedule request.
type ScheduleTrigger struct {
	Name string
}
