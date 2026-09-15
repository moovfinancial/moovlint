package blankdiscard

import (
	"context"
	"database/sql"
)

func badDiscard(err error) {
	_ = err // want "error return blank-discarded without justification"
}

func badTuple() {
	_, _ = doWork() // want "error return blank-discarded without justification"
}

func badSQLExec(db *sql.DB, ctx context.Context) {
	_, _ = db.ExecContext(ctx, "UPDATE t SET x = 1") // want "error return blank-discarded without justification"
}

func goodDiscardWithComment(err error) {
	_ = err // safe to ignore: caller retries on failure
}

func goodTupleWithComment() {
	_, _ = doWork() // results intentionally unused, no error is possible
}

func goodNoDiscard() error {
	_, err := doWork()
	return err
}

func badResultOnly(result func() sql.Result) {
	_ = result() // want "sql.Result blank-discarded without justification"
}

func doWork() (string, error) { return "", nil }
