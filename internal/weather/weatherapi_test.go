package weather_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

const weatherAPIFixture = `{
  "current": {
    "temp_c": 21.5, "temp_f": 70.7,
    "feelslike_c": 20.1, "feelslike_f": 68.2,
    "humidity": 55,
    "wind_kph": 14.4, "wind_mph": 8.9,
    "wind_degree": 90,
    "uv": 5,
    "condition": {"text": " Sunny "}
  },
  "forecast": {
    "forecastday": [
      {
        "date": "2026-07-03",
        "day": {
          "maxtemp_c": 28, "maxtemp_f": 82.4,
          "mintemp_c": 17, "mintemp_f": 62.6,
          "condition": {"text": "Heavy snow"}
        },
        "astro": {"sunrise": "05:43 AM", "sunset": "09:30 PM"}
      }
    ]
  }
}`

func TestParseWeatherAPI(t *testing.T) {
	info, err := weather.ParseWeatherAPI([]byte(weatherAPIFixture), config.Config{Forecast: 1})
	if err != nil {
		t.Fatalf("ParseWeatherAPI: %v", err)
	}
	if info.Description != "Sunny" {
		t.Errorf("Description = %q, want %q", info.Description, "Sunny")
	}
	if info.Type != weather.Sunny {
		t.Errorf("Type = %v, want Sunny", info.Type)
	}
	if info.FeelsLikeC != 20.1 || info.Humidity != 55 || info.WindKph != 14.4 {
		t.Errorf("unexpected extras: feels=%v humidity=%v wind=%v", info.FeelsLikeC, info.Humidity, info.WindKph)
	}
	if info.WindDegree != 90 {
		t.Errorf("WindDegree = %v, want 90", info.WindDegree)
	}
	if info.Sunrise != "05:43 AM" || info.Sunset != "09:30 PM" {
		t.Errorf("Sun = %q/%q, want 05:43 AM/09:30 PM", info.Sunrise, info.Sunset)
	}
	if len(info.Forecast) != 1 {
		t.Fatalf("len(Forecast) = %d, want 1", len(info.Forecast))
	}
	if info.Forecast[0].Type != weather.Snowy {
		t.Errorf("forecast Type = %v, want Snowy", info.Forecast[0].Type)
	}
}

func TestParseWeatherAPIHourly(t *testing.T) {
	// Hour filtering uses real epochs, so build the fixture around now.
	now := time.Now()
	inWindow := now.Add(2 * time.Hour)
	outWindow := now.Add(30 * time.Hour)
	body := fmt.Sprintf(`{
	  "current": {"temp_c": 20, "temp_f": 68, "condition": {"text": "Clear"}},
	  "forecast": {"forecastday": [{
	    "date": "2026-07-02",
	    "day": {"condition": {"text": "Clear"}},
	    "hour": [
	      {"time_epoch": %d, "time": "%s", "temp_c": 22, "temp_f": 71.6, "condition": {"text": "Sunny"}},
	      {"time_epoch": %d, "time": "%s", "temp_c": 11, "temp_f": 51.8, "condition": {"text": "Fog"}}
	    ]
	  }]}
	}`,
		inWindow.Unix(), inWindow.Format("2006-01-02 15:04"),
		outWindow.Unix(), outWindow.Format("2006-01-02 15:04"))

	info, err := weather.ParseWeatherAPI([]byte(body), config.Config{Hourly: true})
	if err != nil {
		t.Fatalf("ParseWeatherAPI: %v", err)
	}
	if len(info.Hourly) != 1 {
		t.Fatalf("len(Hourly) = %d, want 1 (out-of-window point dropped)", len(info.Hourly))
	}
	p := info.Hourly[0]
	if p.TempC != 22 || p.Description != "Sunny" {
		t.Errorf("hourly point = %+v, want 22°C Sunny", p)
	}
	if p.Time.Format("15:04") != inWindow.Format("15:04") {
		t.Errorf("hourly label = %s, want %s", p.Time.Format("15:04"), inWindow.Format("15:04"))
	}
}

func TestBuildWeatherAPIURL(t *testing.T) {
	got := weather.BuildWeatherAPIURL(config.Config{City: "Tokyo", APIKey: "k123"})
	if !strings.HasPrefix(got, "https://api.weatherapi.com/v1/current.json?") {
		t.Errorf("expected current.json endpoint, got %q", got)
	}
	if !strings.Contains(got, "key=k123") || !strings.Contains(got, "q=Tokyo") {
		t.Errorf("missing query params in %q", got)
	}

	got = weather.BuildWeatherAPIURL(config.Config{City: "Tokyo", APIKey: "k123", Forecast: 3})
	if !strings.HasPrefix(got, "https://api.weatherapi.com/v1/forecast.json?") {
		t.Errorf("expected forecast.json endpoint, got %q", got)
	}
	if !strings.Contains(got, "days=3") {
		t.Errorf("missing days param in %q", got)
	}

	// hourly mode needs 2 days of hour data even without -f
	got = weather.BuildWeatherAPIURL(config.Config{City: "Tokyo", APIKey: "k123", Hourly: true})
	if !strings.Contains(got, "days=2") {
		t.Errorf("hourly URL should request 2 days, got %q", got)
	}
}
