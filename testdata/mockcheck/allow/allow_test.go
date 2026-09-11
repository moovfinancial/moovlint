package allow

type mockBoundary struct{}

func (mockBoundary) Call() {}

type replacement struct{} // want "test replacement 'replacement' implements internal interface 'InternalClient'"

func (replacement) Read() {}

func uses() {
	Use(mockBoundary{})
	UseInternal(replacement{})
}
