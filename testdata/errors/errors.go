package errors

func Flag(err error, flag string) error { return err }

const (
	NotValid         = "not_valid"
	NotSerializable  = "not_serializable"
	NotFound         = "not_found"
	NotUnique        = "not_unique"
	NotAuthorized    = "not_authorized"
	NotAvailable     = "not_available"
	NoAuthentication = "no_authentication"
	NotValidState    = "not_valid_state"
)
