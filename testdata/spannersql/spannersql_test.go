package spannersql

import (
	"fmt"

	"cloud.google.com/go/spanner"
)

func testHelper(id string) spanner.Statement {
	return spanner.NewStatement(fmt.Sprintf("SELECT * FROM Accounts WHERE ID = '%s'", id))
}
