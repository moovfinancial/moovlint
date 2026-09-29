package spanrequired

import (
	"context"

	obsql "github.com/moovfinancial/go-libs/observability/sql"
)

// CardRepository issues database calls, which are auto-instrumented.
type CardRepository struct {
	db *obsql.DB
}

// Exempt: the instrumented SQL wrapper already spans the query.
func (r *CardRepository) GetCard(ctx context.Context, id string) error {
	_, err := r.db.QueryContext(ctx, "SELECT 1", id)
	return err
}
