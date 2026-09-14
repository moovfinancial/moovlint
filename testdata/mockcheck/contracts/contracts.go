package contracts

type Repository interface {
	Read() error
}

func Use(Repository) {}

type Dependencies struct {
	Repository Repository
}

type TransfersClient interface {
	Transfer() error
}

func UseClient(TransfersClient) {}
