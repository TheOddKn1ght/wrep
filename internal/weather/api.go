package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
)

var (
	UserAgent  = "wrep"
	httpClient = &http.Client{Timeout: 30 * time.Second}
	RetryDelay = 500 * time.Millisecond
)

func FetchWeather(ctx context.Context, cfg config.Config) (WeatherInfo, error) {
	switch cfg.APIProvider {
	case config.ProviderWeatherAPI:
		return fetchWeatherAPI(ctx, cfg)
	case config.ProviderOpenMeteo:
		return fetchOpenMeteo(ctx, cfg)
	default:
		return fetchWttr(ctx, cfg)
	}
}

// GetJSON fetches a URL through the cache: fresh cache hits skip the network,
// successful responses are cached, and on failure a stale cache entry is
// served with a warning. Live mode always refetches (that's the point of it).
func GetJSON(ctx context.Context, cfg config.Config, urlStr string) ([]byte, error) {
	cache := newDiskCache(cfg)

	if cache != nil && !cfg.Live {
		if body, ok := cache.Fresh(urlStr); ok {
			if cfg.Verbose && !cfg.Quiet {
				fmt.Fprintln(os.Stderr, "Cache hit:", urlStr)
			}
			return body, nil
		}
	}

	if cfg.Verbose && !cfg.Quiet {
		fmt.Fprintln(os.Stderr, "Requesting:", urlStr)
	}

	body, err := fetchHTTP(ctx, cfg, urlStr)
	if err == nil {
		if cache != nil && json.Valid(body) {
			cache.Put(urlStr, body)
		}
		return body, nil
	}

	// Don't serve stale data when the user interrupted the request.
	if cache != nil && ctx.Err() == nil {
		if body, age, ok := cache.Read(urlStr); ok {
			if !cfg.Quiet {
				fmt.Fprintf(os.Stderr, "wrep: fetch failed (%v); using cached data from %s ago\n",
					err, age.Round(time.Second))
			}
			return body, nil
		}
	}
	return nil, err
}

// fetchHTTP performs the request with a single retry on transient failures
// (network errors, 5xx, truncated bodies).
func fetchHTTP(ctx context.Context, cfg config.Config, urlStr string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(RetryDelay):
			}
			if cfg.Verbose && !cfg.Quiet {
				fmt.Fprintln(os.Stderr, "Retrying:", urlStr)
			}
		}
		body, retryable, err := doRequest(ctx, cfg, urlStr)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retryable || ctx.Err() != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func doRequest(ctx context.Context, cfg config.Config, urlStr string) (body []byte, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
	}
	if err := CheckStatus(resp, cfg.APIProvider); err != nil {
		return nil, false, err
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, true, fmt.Errorf("failed to read response body: %w", err)
	}
	return body, false, nil
}

func CheckStatus(resp *http.Response, provider string) error {
	if provider == config.ProviderWeatherAPI {
		switch resp.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusUnauthorized:
			return errors.New("unauthorized: invalid or missing API key")
		case http.StatusBadRequest:
			return errors.New("bad request: city not provided or invalid")
		case http.StatusForbidden:
			return errors.New("forbidden: API access denied or quota exceeded")
		default:
			return fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
		}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
	}
	return nil
}

func ParseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
}
