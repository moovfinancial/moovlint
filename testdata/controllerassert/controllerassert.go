package controllerassert

import "net/http"

type GoodController struct{}

var _ Controller = (*GoodController)(nil)

type Controller interface {
	AppendRoutes(mux *http.ServeMux)
}

func (c *GoodController) AppendRoutes(mux *http.ServeMux) {}

type BadController struct{} // want "controller BadController has AppendRoutes but no compile-time"

func (c *BadController) AppendRoutes(mux *http.ServeMux) {}
