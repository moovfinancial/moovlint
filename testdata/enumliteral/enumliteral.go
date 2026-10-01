package enumliteral

import "testdata/enumliteral/status"

func takes(s status.Status) {}

func compare(s status.Status, str string) bool {
	if s == "created" { // want `use status.StatusCreated instead of the string literal "created"`
		return true
	}
	if "done" != s { // want `use status.StatusDone instead of the string literal "done"`
		return true
	}
	if s == "other" {
		return true
	}
	if s == status.StatusDone {
		return true
	}
	return str == "created"
}

func switches(s status.Status) {
	switch s {
	case "created": // want `use status.StatusCreated instead of the string literal "created"`
	case status.StatusDone:
	case "other":
	}
}

func assigns() status.Status {
	var x status.Status = "done" // want `use status.StatusDone instead of the string literal "done"`
	_ = x
	takes("created")             // want `use status.StatusCreated instead of the string literal "created"`
	_ = status.Status("created") // want `use status.StatusCreated instead of the string literal "created"`
	_ = []status.Status{"done"}  // want `use status.StatusDone instead of the string literal "done"`
	_ = status.Status("other")
	return "done" // want `use status.StatusDone instead of the string literal "done"`
}
