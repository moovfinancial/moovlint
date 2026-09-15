package ctornilguard

import "errors"

type requiredService struct{ clock Clock }

func NewRequiredService(clock Clock) (*requiredService, error) {
	if clock == nil {
		return nil, errors.New("clock is required")
	}
	return &requiredService{clock: clock}, nil
}

func (s *requiredService) Run() {
	if s.clock != nil { // want "dependency clock is already nil-checked by its constructor"
		s.clock.Now()
	}
	if nil == s.clock { // want "dependency clock is already nil-checked by its constructor"
		return
	}
	if s == nil { // Receiver checks do not check a dependency.
		return
	}
}

type optionalService struct{ clock Clock }

func NewOptionalService(clock Clock) (*optionalService, error) {
	if clock == nil {
		return nil, errors.New("missing clock")
	}
	return &optionalService{clock: clock}, nil
}

func NewEmptyOptionalService() *optionalService { return &optionalService{} }

func (s *optionalService) Run() {
	if s.clock != nil {
		s.clock.Now()
	}
}

type mutableService struct{ clock Clock }

func NewMutableService(clock Clock) (*mutableService, error) {
	if clock == nil {
		return nil, errors.New("missing clock")
	}
	return &mutableService{clock: clock}, nil
}

func (s *mutableService) Close() { s.clock = nil }

func (s *mutableService) Run() {
	if s.clock != nil {
		s.clock.Now()
	}
}

type uncheckedService struct{ clock Clock }

func NewUncheckedService(clock Clock) *uncheckedService {
	if clock == nil {
		_ = errors.New("does not exit")
	}
	return &uncheckedService{clock: clock}
}

func (s *uncheckedService) Run() {
	if s.clock != nil {
		s.clock.Now()
	}
}

type resetService struct{ clock Clock }

func NewResetService(clock Clock) (*resetService, error) {
	if clock == nil {
		return nil, errors.New("missing clock")
	}
	clock = nil
	return &resetService{clock: clock}, nil
}

func (s *resetService) Run() {
	if s.clock == nil {
		return
	}
}

type zeroService struct{ clock Clock }

func NewZeroService(clock Clock) (*zeroService, error) {
	if clock == nil {
		return nil, errors.New("missing clock")
	}
	return &zeroService{clock: clock}, nil
}

func allocateZero() *zeroService { return new(zeroService) }

func (s *zeroService) Run() {
	if s.clock == nil {
		return
	}
}

type addressedService struct{ clock Clock }

func NewAddressedService(clock Clock) (*addressedService, error) {
	if clock == nil {
		return nil, errors.New("missing clock")
	}
	return &addressedService{clock: clock}, nil
}

func (s *addressedService) Dependency() *Clock { return &s.clock }

func (s *addressedService) Run() {
	if s.clock == nil {
		return
	}
}

type shadowService struct{ clock Clock }

func NewShadowService(clock Clock) (*shadowService, error) {
	if clock := Clock(nil); clock == nil {
		return nil, errors.New("checks a different variable")
	}
	return &shadowService{clock: clock}, nil
}

func (s *shadowService) Run() {
	if s.clock == nil {
		return
	}
}
