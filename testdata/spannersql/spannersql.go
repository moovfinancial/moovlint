package spannersql

import (
	"fmt"

	"cloud.google.com/go/spanner"
)

const (
	table     = "Accounts"
	indexHint = "@{FORCE_INDEX=AccountsByName}"
)

func directSprintf(id string) spanner.Statement {
	return spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM Accounts WHERE ID = '%s'", id)} // want `Spanner SQL built with fmt.Sprintf from a non-constant value; pass values in Statement.Params`
}

func constSprintf() spanner.Statement {
	return spanner.Statement{SQL: fmt.Sprintf("SELECT * FROM %s%s WHERE ID = @id", table, indexHint)}
}

func localVar(id string) spanner.Statement {
	sql := fmt.Sprintf("SELECT * FROM Accounts WHERE ID = '%s'", id)
	return spanner.Statement{SQL: sql} // want `Spanner SQL built with fmt.Sprintf`
}

func localVarReassigned(id string) spanner.Statement {
	var sql = "SELECT * FROM Accounts"
	if id != "" {
		sql = fmt.Sprintf("SELECT * FROM Accounts WHERE ID = '%s'", id)
	}
	return spanner.Statement{SQL: sql} // want `Spanner SQL built with fmt.Sprintf`
}

func localVarConst() spanner.Statement {
	sql := fmt.Sprintf("SELECT * FROM %s WHERE ID = @id", table)
	return spanner.Statement{SQL: sql, Params: map[string]any{"id": "x"}}
}

func newStatement(id string) spanner.Statement {
	return spanner.NewStatement(fmt.Sprintf("DELETE FROM Accounts WHERE ID = '%s'", id)) // want `Spanner SQL built with fmt.Sprintf`
}

func literal() spanner.Statement {
	return spanner.NewStatement("SELECT * FROM Accounts")
}

func params(id string) spanner.Statement {
	return spanner.Statement{
		SQL:    "SELECT * FROM Accounts WHERE ID = @id",
		Params: map[string]any{"id": id},
	}
}

func positional(id string) spanner.Statement {
	return spanner.Statement{fmt.Sprintf("SELECT * FROM Accounts WHERE ID = '%s'", id), nil} // want `Spanner SQL built with fmt.Sprintf`
}
