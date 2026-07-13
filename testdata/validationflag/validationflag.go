package validationflag

import (
	"github.com/moovfinancial/errors"
	"github.com/moovfinancial/go-libs/mvalidation"
)

type BadRequest struct {
	Name string
}

func (r BadRequest) Validate() error {
	return mvalidation.ValidateStruct(r) // want "mvalidation.ValidateStruct result must be wrapped with errors.Flag"
}

type GoodRequest struct {
	Name string
}

func (r GoodRequest) Validate() error {
	if err := mvalidation.ValidateStruct(r); err != nil {
		return errors.Flag(err, errors.NotValid)
	}
	return nil
}

func (r GoodRequest) ValidateInline() error {
	return errors.Flag(mvalidation.ValidateStruct(r), errors.NotValid)
}
