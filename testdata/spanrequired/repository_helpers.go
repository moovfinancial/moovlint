package spanrequired

import (
	"context"

	obsql "github.com/moovfinancial/go-libs/observability/sql"
	"github.com/moovfinancial/go-libs/observability/telemetry"
)

type ActivityRepository struct {
	db      *obsql.DB
	pricing pricer
}

// Exempt: the database call is reached through a same-package helper, which
// is followed; parseStatus takes ctx but does no traced work.
func (r *ActivityRepository) GetActivity(ctx context.Context, accountID, id string) error {
	query := buildActivityQuery(accountID)
	return r.queryScanActivity(ctx, query, id)
}

// Exempt: two levels of helpers still end in auto-instrumented work.
func (r *ActivityRepository) ListActivity(ctx context.Context, accountID string) error {
	return r.listActivity(ctx, accountID)
}

func (r *ActivityRepository) listActivity(ctx context.Context, accountID string) error {
	if err := r.queryScanActivity(ctx, buildActivityQuery(accountID)); err != nil {
		return err
	}
	return r.queryScanActivity(ctx, buildActivityQuery(accountID)+" UNION")
}

func (r *ActivityRepository) queryScanActivity(ctx context.Context, query string, args ...any) error {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	for rows.Next() {
		parseStatus(ctx, "settled")
	}
	return rows.Err()
}

func parseStatus(ctx context.Context, s string) string {
	telemetry.SetAttributes(ctx, "status", s)
	return s
}

func buildActivityQuery(accountID string) string { return "SELECT " + accountID }

// Exempt: the helper starts its own span, so its work is already traced.
func (r *ActivityRepository) ArchiveActivity(ctx context.Context, id string) error {
	if err := r.archive(ctx, id); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, "DELETE", id)
	return err
}

func (r *ActivityRepository) archive(ctx context.Context, id string) error {
	ctx, span := telemetry.StartSpan(ctx, "archive-activity")
	defer span.End()
	_, err := r.pricing.Price(ctx, id)
	return err
}

// Still reported: the helper mixes database work with an untraced call.
func (r *ActivityRepository) PriceActivity(ctx context.Context, id string) error { // want "exported method ActivityRepository\\.PriceActivity takes context but does not start a telemetry span"
	return r.priceActivity(ctx, id)
}

func (r *ActivityRepository) priceActivity(ctx context.Context, id string) error {
	if _, err := r.pricing.Price(ctx, id); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, "UPDATE", id)
	return err
}

// Recursive helpers terminate; the cycle adds no work of its own.
func (r *ActivityRepository) WalkActivity(ctx context.Context, id string) error {
	return r.walk(ctx, id, 3)
}

func (r *ActivityRepository) walk(ctx context.Context, id string, depth int) error {
	if depth == 0 {
		_, err := r.db.QueryContext(ctx, "SELECT", id)
		return err
	}
	return r.walk(ctx, id, depth-1)
}

// activityStore is implemented by ActivityRepository in this package, so
// calls through it are followed to that implementation.
type activityStore interface {
	GetActivity(ctx context.Context, accountID, id string) error
	PriceActivity(ctx context.Context, id string) error
}

type ActivityService struct {
	store   activityStore
	pricing pricer
}

// Exempt: both repository calls resolve to database-only implementations.
func (s *ActivityService) GetActivities(ctx context.Context, accountID string, ids []string) error {
	for _, id := range ids {
		if err := s.store.GetActivity(ctx, accountID, id); err != nil {
			return err
		}
	}
	return nil
}

// Still reported: the implementation does untraced work besides the database.
func (s *ActivityService) PriceAll(ctx context.Context, ids []string) error { // want "exported method ActivityService\\.PriceAll takes context but does not start a telemetry span"
	for _, id := range ids {
		if err := s.store.PriceActivity(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
