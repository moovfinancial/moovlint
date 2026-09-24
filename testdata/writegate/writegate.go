package writegate

import (
	"context"
	"net/http"

	"github.com/moovfinancial/events/go/eventing"
	v1 "github.com/moovfinancial/events/go/events/v1"
	obssql "github.com/moovfinancial/go-libs/observability/sql"
)

type Repository struct {
	DB *obssql.DB
}

func (r *Repository) Insert(ctx context.Context, name string) error { // want Insert:"writes DB"
	_, err := r.DB.ExecContext(ctx, "INSERT INTO widgets (name) VALUES ($1)", name)
	return err
}

func (r *Repository) Get(ctx context.Context, id string) error {
	_, err := r.DB.QueryContext(ctx, "SELECT name FROM widgets WHERE id = $1", id)
	return err
}

type Service struct {
	Repo *Repository
	DB   *obssql.DB
}

func (s *Service) CreateWidget(ctx context.Context, name string) error { // want CreateWidget:"writes DB"
	return s.Repo.Insert(ctx, name)
}

type API struct {
	Service *Service
	Repo    *Repository
	Events  eventing.EventHandlerContext
}

func (c *API) CreateWidget(w http.ResponseWriter, r *http.Request) { // want CreateWidget:"writes DB"
	_ = c.Service.CreateWidget(r.Context(), "x") // want "database write CreateWidget must run in an events consumer handler"
}

func (c *API) InsertDirect(w http.ResponseWriter, r *http.Request) { // want InsertDirect:"writes DB"
	_, _ = c.Repo.DB.ExecContext(r.Context(), "INSERT INTO widgets (name) VALUES ($1)", "x") // want "database write ExecContext must run in an events consumer handler"
}

func (c *API) GetWidget(w http.ResponseWriter, r *http.Request) {
	_ = c.Repo.Get(r.Context(), "id")
}

func (h *Handler) EventHandlerContext() eventing.EventHandlerContext { // want EventHandlerContext:"writes DB"
	return func(ctx context.Context, event *v1.Event) error {
		return h.Service.CreateWidget(ctx, event.Name)
	}
}

type Handler struct {
	Service *Service
}

func (h *Handler) HandleWidgetRequested(ctx context.Context, event *v1.Event) error { // want HandleWidgetRequested:"writes DB"
	return h.Service.CreateWidget(ctx, event.Name)
}

func consume(h eventing.EventMessageHandler) {}

func registerConsumer(svc *Service) { // want registerConsumer:"writes DB"
	consume(func(ctx context.Context, events []*eventing.EventMessage) error {
		return svc.CreateWidget(ctx, "from-consumer")
	})
}

func httpClosure(svc *Service) http.HandlerFunc { // want httpClosure:"writes DB"
	return func(w http.ResponseWriter, r *http.Request) {
		_ = svc.CreateWidget(r.Context(), "x") // want "database write CreateWidget must run in an events consumer handler"
	}
}

func (c *API) TxWrite(w http.ResponseWriter, r *http.Request) { // want TxWrite:"writes DB"
	_ = c.Repo.DB.InTxScope(r.Context(), nil, func(tx *obssql.Tx) error { // want "database write InTxScope must run in an events consumer handler"
		_, err := tx.ExecContext(r.Context(), "INSERT INTO widgets (name) VALUES ($1)", "x") // want "database write ExecContext must run in an events consumer handler"
		return err
	})
}

type RepositoryIface interface {
	Insert(ctx context.Context, name string) error // want Insert:"writes DB"
}

func (c *API) CreateViaIface(w http.ResponseWriter, r *http.Request, repo RepositoryIface) { // want CreateViaIface:"writes DB"
	_ = repo.Insert(r.Context(), "x") // want "database write Insert must run in an events consumer handler"
}

type recorder struct{ http.ResponseWriter }

func (c *API) CreateViaRecorder(w recorder, r *http.Request) { // want CreateViaRecorder:"writes DB"
	_ = c.Service.CreateWidget(r.Context(), "x") // want "database write CreateWidget must run in an events consumer handler"
}
