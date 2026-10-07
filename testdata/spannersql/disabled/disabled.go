package disabled

import (
	"fmt"

	"cloud.google.com/go/spanner"
)

func directSprintf(id string) spanner.Statement {
	return spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Accounts WHERE ID = '%s'", id)}
}
