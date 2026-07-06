package spanrequired

import (
	"context"

	"github.com/moovfinancial/go-libs/observability/telemetry"
)

type AccountService struct{}

func (s *AccountService) GetAccount(ctx context.Context, id string) (string, error) {
	ctx, _ = telemetry.StartSpan(ctx, "get_account")
	return id, nil
}
