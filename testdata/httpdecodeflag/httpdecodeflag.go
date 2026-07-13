package httpdecodeflag

import (
	"encoding/json"
	"net/http"

	"github.com/moovfinancial/errors"
)

type Handler struct{}

func (h *Handler) BadDecode(w http.ResponseWriter, r *http.Request) error {
	var req struct{ Name string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil { // want "request body decode error must be wrapped with errors.Flag"
		return err
	}
	return nil
}

func (h *Handler) GoodDecode(w http.ResponseWriter, r *http.Request) error {
	var req struct{ Name string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return errors.Flag(err, errors.NotSerializable)
	}
	return nil
}
