package midusage

import (
	"github.com/moovfinancial/go-libs/mid"
)

type Service struct{}

func (s *Service) BadMustParse(id string) {
	_ = mid.MustParseID[mid.Account](id) // want "mid.MustParseID must not be used in production code"
}

func (s *Service) OKParse(id string) {
	_, _ = mid.ParseID[mid.Account](id)
}
