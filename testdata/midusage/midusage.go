package midusage

import (
	"github.com/moovfinancial/go-libs/mid"
)

type Service struct{}

type AccountID = mid.ID[mid.Account]

func (s *Service) BadMustParse(id string) {
	_ = mid.MustParseID[mid.Account](id) // want "mid.MustParseID must not be used in production code"
}

func (s *Service) OKParse(id string) {
	_, _ = mid.ParseID[mid.Account](id)
}

func (s *Service) BadEqual(first, second mid.ID[mid.Account]) bool {
	return first == second // want "mid.ID values must use Equals instead of == or !="
}

func (s *Service) BadEqualAlias(first, second AccountID) bool {
	return second == first // want "mid.ID values must use Equals instead of == or !="
}

func (s *Service) BadNotEqual(first, second mid.ID[mid.Account]) bool {
	return first != second // want "mid.ID values must use Equals instead of == or !="
}

func (s *Service) OKEqual(first, second mid.ID[mid.Account]) bool {
	return first.Equals(second)
}

func (s *Service) OKNotEqual(first, second mid.ID[mid.Account]) bool {
	return !first.Equals(second)
}

func (s *Service) OKStringEqual(first, second string) bool {
	return first == second
}
