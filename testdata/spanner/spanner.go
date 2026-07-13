package spanner

import (
	"context"
	"time"
)

type Client struct{}

func (c *Client) Apply(ctx context.Context, muts []*Mutation) (any, error) {
	return nil, nil
}

func (c *Client) ReadWriteTransaction(ctx context.Context, f func(context.Context, *ReadWriteTransaction) error) (time.Time, error) {
	return time.Time{}, nil
}

type Mutation struct{}

type Row struct{}

type Statement struct {
	SQL    string
	Params map[string]any
}

type Key []any

type ReadWriteTransaction struct{}

func ErrCode(err error) string { return "" }

func Insert(table string, columns []string, values []any) Mutation { return Mutation{} }
func InsertMap(table string, m map[string]any) Mutation            { return Mutation{} }
func InsertOrUpdateStruct(table string, s any) (Mutation, error)   { return Mutation{}, nil }
func InsertStruct(table string, s any) (Mutation, error)           { return Mutation{}, nil }

type NullString struct {
	StringVal string
	Valid     bool
}

type NullJSON struct {
	Value any
	Valid bool
}
