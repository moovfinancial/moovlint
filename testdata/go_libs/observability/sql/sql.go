package sql

import (
	"context"
	gosql "database/sql"
)

type DB struct{}

type Tx struct{}

type TxOptions struct{}

func (d *DB) Exec(query string, args ...any) (gosql.Result, error) { return nil, nil }

func (d *DB) ExecContext(ctx context.Context, query string, args ...any) (gosql.Result, error) {
	return nil, nil
}

func (d *DB) ExecContextRetryable(ctx context.Context, retryParams any, args ...any) (gosql.Result, error) {
	return nil, nil
}

func (d *DB) QueryContext(ctx context.Context, query string, args ...any) (*gosql.Rows, error) {
	return nil, nil
}

func (d *DB) InTxScope(ctx context.Context, opts *TxOptions, f func(sqlTx *Tx) error) error {
	return f(&Tx{})
}

func (t *Tx) Exec(query string, args ...any) (gosql.Result, error) { return nil, nil }

func (t *Tx) ExecContext(ctx context.Context, query string, args ...any) (gosql.Result, error) {
	return nil, nil
}
