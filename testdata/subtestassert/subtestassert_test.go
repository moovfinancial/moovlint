package subtestassert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type suite struct {
	Assert *require.Assertions
}

func TestBadOuterSuite(t *testing.T) {
	s := &suite{Assert: require.New(t)}
	t.Run("inner", func(t *testing.T) {
		s.Assert.NoError(nil) // want "assertion object from the outer test is used inside t.Run"
	})
}

func TestBadOuterRequireNew(t *testing.T) {
	r := require.New(t)
	t.Run("inner", func(t *testing.T) {
		r.NoError(nil) // want "assertion object from the outer test is used inside t.Run"
	})
}

func TestOKLocalRequireNew(t *testing.T) {
	t.Run("inner", func(t *testing.T) {
		r := require.New(t)
		r.NoError(nil)
	})
}

func TestOKPackageLevel(t *testing.T) {
	t.Run("inner", func(t *testing.T) {
		require.NoError(t, nil)
	})
}

func TestOKNestedSubtests(t *testing.T) {
	r := require.New(t)
	_ = r
	t.Run("outer", func(t *testing.T) {
		t.Run("inner", func(t *testing.T) {
			r2 := require.New(t)
			r2.NoError(nil)
		})
	})
}
