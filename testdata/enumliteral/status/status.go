package status

type Status string

const (
	StatusCreated Status = "created"
	StatusDone    Status = "done"
)

func IsDone(s Status) bool {
	return s == "done" // want `use StatusDone instead of the string literal "done"`
}

func Compare(s Status) bool {
	return s == StatusCreated
}
