package weather_test

import (
	"testing"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

const wttrFixture = `{
  "current_condition": [{
    "temp_C": "21", "temp_F": "70",
    "FeelsLikeC": "19", "FeelsLikeF": "66",
    "humidity": "40",
    "windspeedKmph": "12", "windspeedMiles": "7",
    "winddirDegree": "315",
    "uvIndex": "4",
    "localObsDateTime": "2026-07-02 01:00 PM",
    "weatherDesc": [{"value": "Partly cloudy "}]
  }],
  "weather": [
    {
      "date": "2026-07-02",
      "maxtempC": "25", "maxtempF": "77",
      "mintempC": "15", "mintempF": "59",
      "astronomy": [{"sunrise": "06:43 AM", "sunset": "09:12 PM"}],
      "hourly": [
        {"time": "0",    "tempC": "16", "tempF": "61", "weatherDesc": [{"value": "Clear"}]},
        {"time": "300",  "tempC": "15", "tempF": "59", "weatherDesc": [{"value": "Clear"}]},
        {"time": "600",  "tempC": "17", "tempF": "63", "weatherDesc": [{"value": "Sunny"}]},
        {"time": "900",  "tempC": "20", "tempF": "68", "weatherDesc": [{"value": "Sunny"}]},
        {"time": "1200", "tempC": "23", "tempF": "73", "weatherDesc": [{"value": "Light rain"}]},
        {"time": "1500", "tempC": "25", "tempF": "77", "weatherDesc": [{"value": "Sunny"}]},
        {"time": "1800", "tempC": "22", "tempF": "72", "weatherDesc": [{"value": "Clear"}]},
        {"time": "2100", "tempC": "18", "tempF": "64", "weatherDesc": [{"value": "Clear"}]}
      ]
    }
  ]
}`

func TestBuildWttrURL(t *testing.T) {
	got, err := weather.BuildWttrURL(config.Config{City: "New York"})
	if err != nil {
		t.Fatal(err)
	}
	want := "https://wttr.in/New%20York?format=j1"
	if got != want {
		t.Errorf("BuildWttrURL = %q, want %q", got, want)
	}
}

func TestParseWttr(t *testing.T) {
	info, err := weather.ParseWttr([]byte(wttrFixture), config.Config{Forecast: 1})
	if err != nil {
		t.Fatalf("ParseWttr: %v", err)
	}
	if info.Description != "Partly cloudy" {
		t.Errorf("Description = %q, want %q", info.Description, "Partly cloudy")
	}
	if info.TempC != 21 || info.TempF != 70 {
		t.Errorf("Temp = %v/%v, want 21/70", info.TempC, info.TempF)
	}
	if info.FeelsLikeC != 19 || info.FeelsLikeF != 66 {
		t.Errorf("FeelsLike = %v/%v, want 19/66", info.FeelsLikeC, info.FeelsLikeF)
	}
	if info.Humidity != 40 {
		t.Errorf("Humidity = %v, want 40", info.Humidity)
	}
	if info.WindKph != 12 || info.WindMph != 7 {
		t.Errorf("Wind = %v/%v, want 12/7", info.WindKph, info.WindMph)
	}
	if info.WindDegree != 315 {
		t.Errorf("WindDegree = %v, want 315", info.WindDegree)
	}
	if info.Sunrise != "06:43 AM" || info.Sunset != "09:12 PM" {
		t.Errorf("Sun = %q/%q, want 06:43 AM/09:12 PM", info.Sunrise, info.Sunset)
	}
	if info.Type != weather.Cloudy {
		t.Errorf("Type = %v, want Cloudy", info.Type)
	}
	if len(info.Forecast) != 1 {
		t.Fatalf("len(Forecast) = %d, want 1", len(info.Forecast))
	}
	day := info.Forecast[0]
	// index 4 is the noon slot in wttr.in's 3-hourly data
	if day.Description != "Light rain" {
		t.Errorf("forecast Description = %q, want %q (noon slot)", day.Description, "Light rain")
	}
	if day.Type != weather.Rainy {
		t.Errorf("forecast Type = %v, want Rainy", day.Type)
	}
	if day.MinTempC != 15 || day.MaxTempC != 25 {
		t.Errorf("forecast temps = %v..%v, want 15..25", day.MinTempC, day.MaxTempC)
	}
	if day.Date.Format("2006-01-02") != "2026-07-02" {
		t.Errorf("forecast Date = %v, want 2026-07-02", day.Date)
	}
}

func TestParseWttrHourly(t *testing.T) {
	info, err := weather.ParseWttr([]byte(wttrFixture), config.Config{Hourly: true})
	if err != nil {
		t.Fatalf("ParseWttr: %v", err)
	}
	// city-local "now" is 13:00; the window [11:30, next day 13:00] keeps
	// 12:00, 15:00, 18:00 and 21:00
	if len(info.Hourly) != 4 {
		t.Fatalf("len(Hourly) = %d, want 4: %+v", len(info.Hourly), info.Hourly)
	}
	first := info.Hourly[0]
	if first.Time.Format("15:04") != "12:00" {
		t.Errorf("first hourly time = %s, want 12:00", first.Time.Format("15:04"))
	}
	if first.Description != "Light rain" || first.Type != weather.Rainy {
		t.Errorf("first hourly = %q/%v, want Light rain/Rainy", first.Description, first.Type)
	}
	if first.TempC != 23 {
		t.Errorf("first hourly TempC = %v, want 23", first.TempC)
	}
}

func TestParseWttrNoOptionalData(t *testing.T) {
	info, err := weather.ParseWttr([]byte(wttrFixture), config.Config{})
	if err != nil {
		t.Fatalf("ParseWttr: %v", err)
	}
	if len(info.Forecast) != 0 {
		t.Errorf("len(Forecast) = %d, want 0 when Forecast=0", len(info.Forecast))
	}
	if len(info.Hourly) != 0 {
		t.Errorf("len(Hourly) = %d, want 0 when Hourly=false", len(info.Hourly))
	}
}

func TestParseWttrErrors(t *testing.T) {
	cases := map[string]string{
		"invalid JSON":         `{not json`,
		"no current condition": `{"current_condition": []}`,
		"no description":       `{"current_condition": [{"temp_C": "1", "weatherDesc": []}]}`,
	}
	for name, body := range cases {
		if _, err := weather.ParseWttr([]byte(body), config.Config{}); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestWttrCityNowFallback(t *testing.T) {
	if got := weather.WttrCityNow("2026-07-02 01:00 PM"); got.Hour() != 13 {
		t.Errorf("WttrCityNow hour = %d, want 13", got.Hour())
	}
	// unparseable input falls back to the local clock — just ensure it's not zero
	if weather.WttrCityNow("garbage").IsZero() {
		t.Error("WttrCityNow fallback returned zero time")
	}
}
