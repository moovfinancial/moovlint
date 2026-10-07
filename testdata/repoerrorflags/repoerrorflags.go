package repoerrorflags

import (
	"context"
	"database/sql"
	"fmt"

	"cloud.google.com/go/spanner"
	"github.com/moovfinancial/errors"
	"github.com/moovfinancial/go-libs/observability/telemetry"
	"google.golang.org/grpc/codes"
)

type repo struct {
	client *spanner.Client
}

func (r *repo) BadAlreadyExists(ctx context.Context) error {
	_, err := r.client.Apply(ctx, nil)
	if spanner.ErrCode(err) == codes.AlreadyExists { // want "database error check should be flagged with errors.NotUnique"
		return fmt.Errorf("duplicate: %w", err)
	}
	return nil
}

func (r *repo) GoodAlreadyExists(ctx context.Context) error {
	_, err := r.client.Apply(ctx, nil)
	if spanner.ErrCode(err) == codes.AlreadyExists {
		return telemetry.RecordError(ctx, errors.Flag(fmt.Errorf("duplicate: %w", err), errors.NotUnique))
	}
	return nil
}

func (r *repo) BadNotFound(ctx context.Context) error {
	var row *spanner.Row
	if spanner.ErrCode(fmt.Errorf("not found")) == codes.NotFound { // want "database error check should be flagged with errors.NotFound"
		return fmt.Errorf("missing: %w", fmt.Errorf("not found"))
	}
	_ = row
	return nil
}

func (r *repo) GoodNotFound(ctx context.Context) error {
	if spanner.ErrCode(fmt.Errorf("not found")) == codes.NotFound {
		return errors.Flag(fmt.Errorf("missing"), errors.NotFound)
	}
	return nil
}

func (r *repo) OKNotEqual(ctx context.Context) error {
	_, err := r.client.Apply(ctx, nil)
	if spanner.ErrCode(err) != codes.NotFound {
		return fmt.Errorf("apply: %w", err)
	}
	return nil
}

func (r *repo) OKRecoverCreate(ctx context.Context) error {
	_, err := r.client.Apply(ctx, nil)
	if spanner.ErrCode(err) == codes.NotFound {
		_, cerr := r.client.Apply(ctx, nil)
		return cerr
	}
	return nil
}

func (r *repo) OKRecoverNil(ctx context.Context) (*spanner.Row, error) {
	_, err := r.client.Apply(ctx, nil)
	if spanner.ErrCode(err) == codes.NotFound {
		return nil, nil
	}
	return nil, err
}

func BadNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) { // want "database error check should be flagged with errors.NotFound"
		return fmt.Errorf("load: %w", err)
	}
	return nil
}

func GoodNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return errors.Flag(err, errors.NotFound)
	}
	return nil
}
