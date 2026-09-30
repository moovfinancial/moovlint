package spannertxcapture

import (
	"context"

	"cloud.google.com/go/spanner"
)

func load() []string { return nil }

func appendNoReset(ctx context.Context, c *spanner.Client) {
	var results []string
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		results = append(results, "a") // want `results is written inside the Spanner transaction closure without a reset`
		results = append(results, "b")
		return nil
	})
	_ = results
}

func appendWithReset(ctx context.Context, c *spanner.Client) {
	var results, reused []string
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		results = nil
		reused = reused[:0]
		for _, s := range load() {
			results = append(results, s)
			reused = append(reused, s)
		}
		return nil
	})
	_, _ = results, reused
}

func conditionalNoReset(ctx context.Context, c *spanner.Client, cond bool) {
	found := false
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		if cond {
			found = true // want `found is written inside the Spanner transaction closure without a reset`
		}
		return nil
	})
	_ = found
}

func conditionalWithReset(ctx context.Context, c *spanner.Client, cond bool) {
	found := false
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		found = false
		if cond {
			found = true
		}
		return nil
	})
	_ = found
}

func unconditional(ctx context.Context, c *spanner.Client) {
	var results []string
	var n int
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		results = load()
		local := 0
		local++
		_ = n
		return nil
	})
	_ = results
}

func mapWrite(ctx context.Context, c *spanner.Client) {
	seen := map[string]bool{}
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		seen["a"] = true // want `seen is written inside the Spanner transaction closure without a reset`
		return nil
	})
	_ = seen
}

func mapWithReset(ctx context.Context, c *spanner.Client) {
	seen := map[string]bool{}
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		seen = make(map[string]bool)
		seen["a"] = true
		return nil
	})
	_ = seen
}

func counter(ctx context.Context, c *spanner.Client) {
	count, total := 0, 0
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		count++    // want `count is written inside the Spanner transaction closure without a reset`
		total += 2 // want `total is written inside the Spanner transaction closure without a reset`
		return nil
	})
	_, _ = count, total
}

func nestedFuncLit(ctx context.Context, c *spanner.Client) {
	count := 0
	_, _ = c.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		f := func() { count++ }
		_ = f
		return nil
	})
	_ = count
}

func withOptions(ctx context.Context, c *spanner.Client) {
	var results []string
	_, _ = c.ReadWriteTransactionWithOptions(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		results = append(results, "a") // want `results is written inside the Spanner transaction closure without a reset`
		return nil
	}, spanner.TransactionOptions{})
	_ = results
}

type repo struct{ client *spanner.Client }

func (r *repo) ReadWriteTransaction(ctx context.Context, f func(context.Context, *spanner.ReadWriteTransaction) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, f)
	return err
}

func wrapper(ctx context.Context, r *repo) {
	count := 0
	_ = r.ReadWriteTransaction(ctx, func(ctx context.Context, tx *spanner.ReadWriteTransaction) error {
		count++ // want `count is written inside the Spanner transaction closure without a reset`
		return nil
	})
	_ = count
}

type other struct{}

func (other) ReadWriteTransaction(ctx context.Context, f func(context.Context) error) error {
	return f(ctx)
}

func capturedErr(ctx context.Context, c *spanner.Client, cond bool) error {
	var err error
	_, txErr := c.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		if cond {
			_, err = txn.Update(ctx, spanner.Statement{})
		}
		return err
	})
	return txErr
}

func notSpanner(ctx context.Context, o other) {
	count := 0
	_ = o.ReadWriteTransaction(ctx, func(ctx context.Context) error {
		count++
		return nil
	})
	_ = count
}
