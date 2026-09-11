package allow

type AllowedBoundary interface{ Call() }
type InternalClient interface{ Read() }

func Use(AllowedBoundary)        {}
func UseInternal(InternalClient) {}
