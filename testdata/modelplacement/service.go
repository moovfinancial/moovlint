package modelplacement

type CreateRequest struct { // want "data model CreateRequest is declared in service.go"
	Name string
}

func (CreateRequest) Validate() error { return nil }

type service struct{ name string }

type Service struct{ Run func() }

type Request interface{ Name() string }

type Config struct{ Topic string }

type StatefulRequest struct{ Name string }

func (StatefulRequest) Run() {}

type AliasRequest = CreateRequest

func local() {
	type LocalRequest struct{ Name string }
	_ = LocalRequest{}
}
