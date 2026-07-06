package log

type Valuer interface {
	Value() any
}

func String(s string) Valuer { return valuer{s} }

type valuer struct{ v any }

func (v valuer) Value() any { return v.v }

type Logger interface {
	Set(key string, value Valuer) Logger
	With(ctxs ...Context) Logger
	Debug() Logger
	Info() Logger
	Warn() Logger
	Error() Logger
	Fatal() Logger
	Log(message string)
	Logf(format string, args ...any)
	Send()
	LogError(error) LoggedError
	LogErrorf(format string, args ...any) LoggedError
}

type Context interface{ Context() map[string]Valuer }

type LoggedError interface{ error }

type Fields map[string]Valuer

func (f Fields) Context() map[string]Valuer { return f }
