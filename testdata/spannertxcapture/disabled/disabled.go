package disabled

import (
	"context"

	"cloud.google.com/go/spanner"
)

func appendNoReset(ctx context.Context, c *spanner.Client) {
	var results []string
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		results = append(results, "a")
		return nil
	})
	_ = results
}
