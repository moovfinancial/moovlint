package wrapnil

import (
	"errors"
	"fmt"
)

type thing struct{}

func load() (*thing, error) { return nil, nil }

func someWrap(err error) error { return err }

func orErrFirst() error {
	v, err := load()
	if err != nil || v == nil {
		return fmt.Errorf("load: %w", err) // want `%w wraps err, but err can be nil here because the if condition also passes when err == nil; handle that case separately`
	}
	return nil
}

func orErrSecond() error {
	v, err := load()
	if v == nil || err != nil {
		return fmt.Errorf("load: %w", err) // want `%w wraps err`
	}
	return nil
}

func orDeepChain(ok bool) error {
	v, err := load()
	if (!ok || v == nil) || (err != nil) {
		return fmt.Errorf("load: %w", err) // want `%w wraps err`
	}
	return nil
}

func errAlone() error {
	_, err := load()
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	return nil
}

func andChain() error {
	v, err := load()
	if err != nil && v == nil {
		return fmt.Errorf("load: %w", err)
	}
	return nil
}

func percentV() error {
	v, err := load()
	if err != nil || v == nil {
		return fmt.Errorf("load: %v", err)
	}
	return nil
}

func nestedRecheck() error {
	v, err := load()
	if err != nil || v == nil {
		if err != nil {
			return fmt.Errorf("load: %w", err)
		}
		return errors.New("not found")
	}
	return nil
}

func otherErr() error {
	other := errors.New("other")
	v, err := load()
	if err != nil || v == nil {
		return fmt.Errorf("load: %w", other)
	}
	return nil
}

func twoVerbsWrapSecond(name string) error {
	v, err := load()
	if err != nil || v == nil {
		return fmt.Errorf("%s: %w", name, err) // want `%w wraps err`
	}
	return nil
}

func twoVerbsErrFirst() error {
	v, err := load()
	if err != nil || v == nil {
		return fmt.Errorf("%s: %w", err, errors.New("x"))
	}
	return nil
}

func escapedPercent() error {
	v, err := load()
	if err != nil || v == nil {
		return fmt.Errorf("100%%: %w", err) // want `%w wraps err`
	}
	return nil
}

func wrappedCall() error {
	v, err := load()
	if err != nil || v == nil {
		return someWrap(fmt.Errorf("x: %w", err)) // want `%w wraps err`
	}
	return nil
}

func funcLit() error {
	v, err := load()
	if err != nil || v == nil {
		f := func() error { return fmt.Errorf("x: %w", err) }
		return f()
	}
	return nil
}

func nonConstFormat(format string) error {
	v, err := load()
	if err != nil || v == nil {
		return fmt.Errorf(format, err)
	}
	return nil
}
