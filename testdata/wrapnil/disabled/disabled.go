package disabled

import "fmt"

func load() (*int, error) { return nil, nil }

func orChain() error {
	v, err := load()
	if err != nil || v == nil {
		return fmt.Errorf("load: %w", err)
	}
	return nil
}
