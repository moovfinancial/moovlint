package uuid

import "time"

type UUID [16]byte

func New() UUID                                    { return UUID{} }
func NewString() string                            { return "" }
func NewRandom() (UUID, error)                     { return UUID{}, nil }
func NewRandomFromTime(t time.Time) (UUID, error)  { return UUID{}, nil }
func NewV6() (UUID, error)                         { return UUID{}, nil }
func NewV7() (UUID, error)                         { return UUID{}, nil }

func (u UUID) String() string { return "" }
