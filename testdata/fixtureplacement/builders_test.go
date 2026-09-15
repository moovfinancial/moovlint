package fixtureplacement

import "testing"

func createRequest() CreateRequest { // want "test helper createRequest builds CreateRequest"
	return CreateRequest{Name: "test"}
}

func pointerRequest() *CreateRequest { // want "test helper pointerRequest builds CreateRequest"
	return &CreateRequest{Name: "test"}
}

func wrapperRequest() CreateRequest { // want "test helper wrapperRequest builds CreateRequest"
	return createRequest()
}

func zeroRequest() CreateRequest { return CreateRequest{} }

func assertRequest(t *testing.T, req CreateRequest) { t.Helper() }

func newService() Service { return Service{Run: func() {}} }

func newPrivate() privateRecord { return privateRecord{Name: "local"} }

func scalarHelper() string { return "name" }

func setupEnvironment() (Service, CreateRequest) {
	return Service{Run: func() {}}, CreateRequest{Name: "setup"}
}

func requestWithError() (CreateRequest, error) { // want "test helper requestWithError builds CreateRequest"
	return CreateRequest{Name: "fixture"}, nil
}
