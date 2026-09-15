package testsleep

import (
	"testing"
	"time"
)

func TestBadSleep(t *testing.T) {
	time.Sleep(100 * time.Millisecond) // want "time.Sleep in tests causes flaky synchronization"
}

func TestOKEventually(t *testing.T) {
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		break
	}
}
