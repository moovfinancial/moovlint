package stime

type TimeService interface {
	Now() any
}

type StaticTimeService struct{}

func NewStaticTimeService() StaticTimeService { return StaticTimeService{} }

func (s StaticTimeService) Now() any { return nil }
