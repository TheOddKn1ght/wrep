package weather_test

import (
	"os"
	"testing"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/weather"
)

// TestMain isolates tests from the real user cache and speeds up retries.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "wrep-test-cache")
	if err != nil {
		panic(err)
	}
	weather.UserCacheDir = func() (string, error) { return dir, nil }
	weather.RetryDelay = time.Millisecond

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
