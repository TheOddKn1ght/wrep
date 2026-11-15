package weather_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

func TestDiskCacheRoundTrip(t *testing.T) {
	c := &weather.DiskCache{Dir: t.TempDir(), TTL: time.Minute}
	url := "https://example.com/weather"

	if _, ok := c.Fresh(url); ok {
		t.Fatal("Fresh() on empty cache should miss")
	}

	c.Put(url, []byte(`{"hello": "world"}`))

	body, ok := c.Fresh(url)
	if !ok {
		t.Fatal("Fresh() should hit after Put")
	}
	if string(body) != `{"hello": "world"}` {
		t.Errorf("cached body = %q", body)
	}

	// different URL must not collide
	if _, ok := c.Fresh("https://example.com/other"); ok {
		t.Error("different URL should miss")
	}
}

func TestDiskCacheTTLExpiry(t *testing.T) {
	c := &weather.DiskCache{Dir: t.TempDir(), TTL: time.Minute}
	url := "https://example.com/weather"
	c.Put(url, []byte(`{}`))

	// Backdate the entry beyond the TTL.
	old := time.Now().Add(-2 * time.Minute)
	if err := os.Chtimes(c.Path(url), old, old); err != nil {
		t.Fatal(err)
	}

	if _, ok := c.Fresh(url); ok {
		t.Error("expired entry should not be fresh")
	}
	if _, age, ok := c.Read(url); !ok || age < time.Minute {
		t.Errorf("Read() should still return stale entry (ok=%v age=%v)", ok, age)
	}
}

func TestGetJSONServesFreshCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{"n": 1}`))
	}))
	defer srv.Close()

	cfg := config.Config{Quiet: true, CacheTTL: time.Minute}
	url := srv.URL + "/cache-fresh-test"

	for i := 0; i < 3; i++ {
		body, err := weather.GetJSON(context.Background(), cfg, url)
		if err != nil {
			t.Fatalf("GetJSON #%d: %v", i, err)
		}
		if string(body) != `{"n": 1}` {
			t.Errorf("body = %q", body)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server calls = %d, want 1 (second and third served from cache)", got)
	}
}

func TestGetJSONNoCacheBypasses(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfg := config.Config{Quiet: true, NoCache: true}
	url := srv.URL + "/no-cache-test"
	for i := 0; i < 2; i++ {
		if _, err := weather.GetJSON(context.Background(), cfg, url); err != nil {
			t.Fatal(err)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server calls = %d, want 2 with -no-cache", got)
	}
}

func TestGetJSONLiveModeSkipsFreshCache(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cfg := config.Config{Quiet: true, Live: true, CacheTTL: time.Hour}
	url := srv.URL + "/live-test"
	for i := 0; i < 2; i++ {
		if _, err := weather.GetJSON(context.Background(), cfg, url); err != nil {
			t.Fatal(err)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server calls = %d, want 2 (live mode must refetch)", got)
	}
}

func TestGetJSONStaleFallback(t *testing.T) {
	var fail atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"cached": true}`))
	}))
	defer srv.Close()

	// Tiny TTL so the primed entry is stale by the second call.
	cfg := config.Config{Quiet: true, CacheTTL: time.Nanosecond}
	url := srv.URL + "/stale-test"

	if _, err := weather.GetJSON(context.Background(), cfg, url); err != nil {
		t.Fatalf("priming call: %v", err)
	}

	fail.Store(true)
	body, err := weather.GetJSON(context.Background(), cfg, url)
	if err != nil {
		t.Fatalf("expected stale fallback, got error: %v", err)
	}
	if string(body) != `{"cached": true}` {
		t.Errorf("stale body = %q", body)
	}
}

func TestGetJSONInvalidJSONNotCached(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Write([]byte(`{truncated`))
	}))
	defer srv.Close()

	cfg := config.Config{Quiet: true, CacheTTL: time.Minute}
	url := srv.URL + "/invalid-json-test"
	for i := 0; i < 2; i++ {
		// GetJSON succeeds (HTTP was fine); parsing fails later at the caller.
		if _, err := weather.GetJSON(context.Background(), cfg, url); err != nil {
			t.Fatal(err)
		}
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server calls = %d, want 2 (invalid JSON must not be cached)", got)
	}
}
