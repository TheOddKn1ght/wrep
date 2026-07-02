package weather

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
)

// UserCacheDir is a seam for tests so they never touch the real cache.
var UserCacheDir = os.UserCacheDir

// DiskCache stores raw provider responses keyed by request URL under the
// user cache directory. Entries newer than TTL are served without a network
// round-trip; older entries are kept as a stale fallback for when the
// provider is unreachable.
type DiskCache struct {
	Dir string
	TTL time.Duration
}

// newDiskCache returns nil when caching is disabled or no cache directory is
// available; callers treat a nil cache as a no-op.
func newDiskCache(cfg config.Config) *DiskCache {
	if cfg.NoCache {
		return nil
	}
	root, err := UserCacheDir()
	if err != nil {
		return nil
	}
	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = config.DefaultCacheTTL
	}
	return &DiskCache{Dir: filepath.Join(root, "wrep"), TTL: ttl}
}

func (c *DiskCache) Path(url string) string {
	sum := sha256.Sum256([]byte(url))
	return filepath.Join(c.Dir, hex.EncodeToString(sum[:])+".json")
}

// Fresh returns the cached body only if it is younger than the TTL.
func (c *DiskCache) Fresh(url string) ([]byte, bool) {
	body, age, ok := c.Read(url)
	if !ok || age > c.TTL {
		return nil, false
	}
	return body, true
}

// Read returns the cached body regardless of age.
func (c *DiskCache) Read(url string) ([]byte, time.Duration, bool) {
	p := c.Path(url)
	fi, err := os.Stat(p)
	if err != nil {
		return nil, 0, false
	}
	body, err := os.ReadFile(p)
	if err != nil {
		return nil, 0, false
	}
	return body, time.Since(fi.ModTime()), true
}

// Put stores a response body; failures are silent since the cache is
// best-effort.
func (c *DiskCache) Put(url string, body []byte) {
	if err := os.MkdirAll(c.Dir, 0o700); err != nil {
		return
	}
	_ = os.WriteFile(c.Path(url), body, 0o600)
}
