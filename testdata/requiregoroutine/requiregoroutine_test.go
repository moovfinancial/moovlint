package requiregoroutine

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBadGoStatement(t *testing.T) {
	go func() {
		require.NoError(t, nil) // want "require/t.Fatal called from a non-test goroutine closure"
	}()
}

func TestBadHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("boom") // want "require/t.Fatal called from a non-test goroutine closure"
	}))
	defer server.Close()
}

func TestBadCallback(t *testing.T) {
	runAsync(func() {
		t.Fatalf("bad: %v", "x") // want "require/t.Fatal called from a non-test goroutine closure"
	})
}

func TestOKSubtest(t *testing.T) {
	t.Run("inner", func(t *testing.T) {
		require.NoError(t, nil)
	})
}

func TestOKDefer(t *testing.T) {
	defer func() {
		require.NoError(t, nil)
	}()
}

func TestOKCleanup(t *testing.T) {
	t.Cleanup(func() {
		require.NoError(t, nil)
	})
}

func TestOKFatalOnTestGoroutine(t *testing.T) {
	if false {
		t.Fatal("unreachable")
	}
}

func runAsync(f func()) { f() }
