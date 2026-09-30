package outert

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type scope struct {
	T *testing.T
}

func TestBadOuterT(t *testing.T) {
	t.Run("inner", func(st *testing.T) {
		require.NoError(t, nil) // want "the outer test's t is used inside t.Run"
		require.NoError(st, nil)
	})
}

func TestBadOuterField(t *testing.T) {
	sc := scope{T: t}
	t.Run("inner", func(t *testing.T) {
		require.NoError(sc.T, nil) // want "the outer test's t is used inside t.Run"
	})
}

func TestBadOuterHelper(t *testing.T) {
	build := func() { require.NoError(t, nil) }
	t.Run("inner", func(st *testing.T) {
		run := func() { t.Helper() } // want "the outer test's t is used inside t.Run"
		run()
		build()
		_ = st
	})
}

func TestOKShadowed(t *testing.T) {
	t.Run("inner", func(t *testing.T) {
		require.NoError(t, nil)
		sc := scope{T: t}
		require.NoError(sc.T, nil)
	})
}

func TestOKNested(t *testing.T) {
	t.Run("outer", func(t *testing.T) {
		t.Run("inner", func(t *testing.T) {
			require.NoError(t, nil)
		})
	})
}

func TestOKBaseCheckStillRuns(t *testing.T) {
	r := require.New(t)
	t.Run("inner", func(t *testing.T) {
		r.NoError(nil) // want "assertion object from the outer test is used inside t.Run"
	})
}
