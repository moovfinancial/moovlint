package disabled

import "fmt"

func bad(id string) error {
	return fmt.Errorf("account %s failed", id)
}
