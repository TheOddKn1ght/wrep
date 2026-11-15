package weather_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

func TestClassifyWeather(t *testing.T) {
	tests := []struct {
		desc string
		want weather.WeatherType
	}{
		{"Sunny", weather.Sunny},
		{"Clear", weather.Sunny},
		{"Partly cloudy", weather.Cloudy},
		{"Overcast", weather.Cloudy},
		{"Light rain", weather.Rainy},
		{"Patchy rain showers", weather.Rainy},
		{"Light drizzle", weather.Rainy},
		{"Moderate snow", weather.Snowy},
		{"Sleet", weather.Snowy},
		{"Blizzard", weather.Snowy},
		{"Slight snow showers", weather.Snowy},
		{"Thundery outbreaks possible", weather.Stormy},
		{"Patchy light rain with thunder", weather.Stormy},
		{"Storm", weather.Stormy},
		{"Fog", weather.Foggy},
		{"Mist", weather.Foggy},
		{"Haze", weather.Foggy},
		{"Blowing widgets", weather.Unknown},
		{"", weather.Unknown},
	}
	for _, tt := range tests {
		if got := weather.ClassifyWeather(tt.desc); got != tt.want {
			t.Errorf("ClassifyWeather(%q) = %v, want %v", tt.desc, got, tt.want)
		}
	}
}

func TestWindCompass(t *testing.T) {
	tests := []struct {
		deg  float64
		want string
	}{
		{0, "N"},
		{90, "E"},
		{180, "S"},
		{270, "W"},
		{315, "NW"},
		{22.5, "NNE"},
		{350, "N"},
		{-45, "NW"},
		{405, "NE"},
	}
	for _, tt := range tests {
		if got := weather.WindCompass(tt.deg); got != tt.want {
			t.Errorf("WindCompass(%v) = %q, want %q", tt.deg, got, tt.want)
		}
	}
}

func TestConversions(t *testing.T) {
	if got := weather.CToF(0); got != 32 {
		t.Errorf("CToF(0) = %v, want 32", got)
	}
	if got := weather.CToF(100); got != 212 {
		t.Errorf("CToF(100) = %v, want 212", got)
	}
	if got := weather.KphToMph(1.609344); got < 0.999 || got > 1.001 {
		t.Errorf("KphToMph(1.609344) = %v, want ~1", got)
	}
}

func TestCheckStatus(t *testing.T) {
	tests := []struct {
		provider string
		code     int
		wantErr  bool
	}{
		{config.ProviderWttr, http.StatusOK, false},
		{config.ProviderWttr, http.StatusNotFound, true},
		{config.ProviderOpenMeteo, http.StatusBadRequest, true},
		{config.ProviderWeatherAPI, http.StatusOK, false},
		{config.ProviderWeatherAPI, http.StatusUnauthorized, true},
		{config.ProviderWeatherAPI, http.StatusBadRequest, true},
		{config.ProviderWeatherAPI, http.StatusForbidden, true},
	}
	for _, tt := range tests {
		resp := &http.Response{StatusCode: tt.code, Status: http.StatusText(tt.code)}
		err := weather.CheckStatus(resp, tt.provider)
		if (err != nil) != tt.wantErr {
			t.Errorf("CheckStatus(%d, %s) error = %v, wantErr %v", tt.code, tt.provider, err, tt.wantErr)
		}
	}
}

func TestParseFloat(t *testing.T) {
	if got := weather.ParseFloat(" 21.5 "); got != 21.5 {
		t.Errorf("ParseFloat(\" 21.5 \") = %v, want 21.5", got)
	}
	if got := weather.ParseFloat("garbage"); got != 0 {
		t.Errorf("ParseFloat(\"garbage\") = %v, want 0", got)
	}
}

func TestFetchWeatherCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := weather.FetchWeather(ctx, config.Config{APIProvider: config.ProviderWttr, City: "London", NoCache: true})
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// TestFetchWeatherEndToEnd exercises the full pipeline (URL build, HTTP,
// User-Agent, parse) against a stub wttr.in.
func TestFetchWeatherEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "wrep") {
			t.Errorf("User-Agent = %q, want wrep prefix", ua)
		}
		if r.URL.Query().Get("format") != "j1" {
			t.Errorf("missing format=j1 in %q", r.URL.String())
		}
		w.Write([]byte(wttrFixture))
	}))
	defer srv.Close()

	oldBase := weather.WttrBase
	weather.WttrBase = srv.URL
	defer func() { weather.WttrBase = oldBase }()

	info, err := weather.FetchWeather(context.Background(), config.Config{APIProvider: config.ProviderWttr, City: "London", NoCache: true})
	if err != nil {
		t.Fatalf("FetchWeather: %v", err)
	}
	if info.Description != "Partly cloudy" {
		t.Errorf("Description = %q, want %q", info.Description, "Partly cloudy")
	}
}

// TestFetchRetriesOn5xx confirms a transient 500 is retried once and then
// succeeds.
func TestFetchRetriesOn5xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(wttrFixture))
	}))
	defer srv.Close()

	oldBase := weather.WttrBase
	weather.WttrBase = srv.URL
	defer func() { weather.WttrBase = oldBase }()

	_, err := weather.FetchWeather(context.Background(), config.Config{APIProvider: config.ProviderWttr, City: "London", NoCache: true, Quiet: true})
	if err != nil {
		t.Fatalf("FetchWeather after retry: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("server calls = %d, want 2 (initial + retry)", got)
	}
}

// TestFetchNoRetryOn4xx confirms client errors fail fast without a retry.
func TestFetchNoRetryOn4xx(t *testing.T) {
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()

	oldBase := weather.WttrBase
	weather.WttrBase = srv.URL
	defer func() { weather.WttrBase = oldBase }()

	_, err := weather.FetchWeather(context.Background(), config.Config{APIProvider: config.ProviderWttr, City: "London", NoCache: true, Quiet: true})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if got := calls.Load(); got != 1 {
		t.Errorf("server calls = %d, want 1 (no retry on 4xx)", got)
	}
}
