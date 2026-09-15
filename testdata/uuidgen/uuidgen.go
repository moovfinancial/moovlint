package uuidgen

import (
	"github.com/google/uuid"
	"github.com/moovfinancial/go-libs/mid"
)

type Service struct{}

func (s *Service) BadNewString() string {
	return uuid.NewString() // want "uuid.NewString generates untyped IDs"
}

func (s *Service) BadNew() uuid.UUID {
	return uuid.New() // want "uuid.New generates untyped IDs"
}

func (s *Service) BadNewV7() (uuid.UUID, error) {
	return uuid.NewV7() // want "uuid.NewV7 generates untyped IDs"
}

func (s *Service) GoodRandomID() mid.ID[mid.Account] {
	return mid.NewRandomID[mid.Account](nil)
}

func (s *Service) GoodParse(id string) (mid.ID[mid.Account], error) {
	return mid.ParseID[mid.Account](id)
}
