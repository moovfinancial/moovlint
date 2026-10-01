package errmsgconst

import (
	"errors"
	"fmt"
)

const table = "Payouts"

func examples(id string, n int, err error) []error {
	return []error{
		fmt.Errorf("account %s failed", id),              // want `error message includes id`
		fmt.Errorf("loading %s: %w", id, err),            // want `error message includes id`
		fmt.Errorf("retry %d of 3", n),                   // want `error message includes n`
		errors.New(fmt.Sprintf("account %s failed", id)), // want `error message includes id`
		fmt.Errorf("loading account: %w", err),
		fmt.Errorf("loading account: %v", err),
		fmt.Errorf("reading %s: %w", table, err),
		fmt.Errorf("100%% done: %w", err),
		errors.New("account failed"),
	}
}

func dynamicFormat(format string, id string) error {
	return fmt.Errorf(format, id)
}
