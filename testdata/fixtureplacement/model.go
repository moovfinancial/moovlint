package fixtureplacement

type CreateRequest struct {
	Name string
}

type Service struct {
	Run func()
}

type privateRecord struct{ Name string }
