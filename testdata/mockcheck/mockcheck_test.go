package mockcheck

import "context"

type mockPayoutService struct{} // want "hand-rolled mock 'mockPayoutService' implements same-package interface 'PayoutService'"

func (m *mockPayoutService) GetPayout(ctx context.Context, id string) (string, error) {
	return "mock-result", nil
}

func (m *mockPayoutService) CreatePayout(ctx context.Context, amount int64) (string, error) {
	return "mock-created", nil
}

type fakeValidator struct{} // want "hand-rolled mock 'fakeValidator' implements same-package interface 'ExternalService'"

func (f *fakeValidator) Validate(ctx context.Context, data string) bool {
	return true
}
