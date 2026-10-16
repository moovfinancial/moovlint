package spanrequired

import (
	"context"

	obsql "github.com/moovfinancial/go-libs/observability/sql"
)

// Test doubles that live next to the code they replace (ex. service_mock.go) are exempt: the span
// belongs to the real method.

type ServiceMock struct {
	hook func(context.Context) error
}

func (m *ServiceMock) GetCard(ctx context.Context, id string) error {
	return m.hook(ctx)
}

func (m *ServiceMock) ApplyForcePost(ctx context.Context, id string) error {
	if err := m.hook(ctx); err != nil {
		return err
	}
	return m.hook(ctx)
}

type MockCardsRepo struct{}

func (m *MockCardsRepo) Decrement(ctx context.Context, id string) error {
	return nil
}

type fakeLedger struct{}

func (f *fakeLedger) Post(ctx context.Context, id string) error {
	return nil
}

// Paystub only ends in "stub", so it is not a test double and is still checked.
type Paystub struct{}

func (h *Paystub) Get(ctx context.Context, id string) error { // want "exported method Paystub\\.Get takes context but does not start a telemetry span"
	return nil
}

// PaymentStore has a real, auto-instrumented implementation and a mock next to it. The mock's
// untraced work must not count against a service that calls the interface.
type PaymentStore interface {
	Save(ctx context.Context, id string) error
}

type sqlPaymentStore struct {
	db *obsql.DB
}

func (s *sqlPaymentStore) Save(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "INSERT 1", id)
	return err
}

type PaymentStoreMock struct {
	hook func(context.Context) error
}

func (m *PaymentStoreMock) Save(ctx context.Context, id string) error {
	return m.hook(ctx)
}

type PaymentService struct {
	store PaymentStore
}

// Exempt: the only implementation that counts is the SQL one, which is auto-instrumented.
func (s *PaymentService) SavePayment(ctx context.Context, id string) error {
	return s.store.Save(ctx, id)
}
