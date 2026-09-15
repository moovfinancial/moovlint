package stime

import "time"

type TimeService interface {
	Now() time.Time
}

type StaticTimeService struct{}

func NewStaticTimeService() StaticTimeService { return StaticTimeService{} }

func (s StaticTimeService) Now() time.Time { return time.Time{} }
