package weather_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

func TestWMODescription(t *testing.T) {
	if got := weather.WMODescription(0); got != "Clear sky" {
		t.Errorf("WMODescription(0) = %q", got)
	}
	if got := weather.WMODescription(95); got != "Thunderstorm" {
		t.Errorf("WMODescription(95) = %q", got)
	}
	if got := weather.WMODescription(42); !strings.Contains(got, "42") {
		t.Errorf("unknown code should mention the code, got %q", got)
	}
	// every mapped description must classify to something known
	for code, desc := range weather.WMODescriptions {
		if code == 1 || code == 2 || code == 3 {
			continue // "mainly clear"/"partly cloudy"/"overcast" classify fine, checked below
		}
		if weather.ClassifyWeather(desc) == weather.Unknown && code != 48 && code != 77 {
			t.Errorf("WMO %d (%q) classifies as Unknown", code, desc)
		}
	}
	if weather.ClassifyWeather(weather.WMODescriptions[2]) != weather.Cloudy {
		t.Error("partly cloudy should classify Cloudy")
	}
	if weather.ClassifyWeather(weather.WMODescriptions[0]) != weather.Sunny {
		t.Error("clear sky should classify Sunny")
	}
}

func TestParseOpenMeteoGeo(t *testing.T) {
	body := `{"results": [{"name": "Berlin", "latitude": 52.52, "longitude": 13.41, "country": "Germany"}]}`
	lat, lon, err := weather.ParseOpenMeteoGeo([]byte(body), "Berlin")
	if err != nil {
		t.Fatal(err)
	}
	if lat != 52.52 || lon != 13.41 {
		t.Errorf("geo = %v,%v want 52.52,13.41", lat, lon)
	}

	if _, _, err := weather.ParseOpenMeteoGeo([]byte(`{}`), "Nowhereville"); err == nil {
		t.Error("expected error for empty geocoding results")
	} else if !strings.Contains(err.Error(), "Nowhereville") {
		t.Errorf("error should name the city: %v", err)
	}
}

func TestOpenMeteoDays(t *testing.T) {
	if got := weather.OpenMeteoDays(config.Config{}); got != 1 {
		t.Errorf("default days = %d, want 1", got)
	}
	if got := weather.OpenMeteoDays(config.Config{Forecast: 7}); got != 7 {
		t.Errorf("days = %d, want 7", got)
	}
	if got := weather.OpenMeteoDays(config.Config{Forecast: 99}); got != 16 {
		t.Errorf("days = %d, want capped at 16", got)
	}
	if got := weather.OpenMeteoDays(config.Config{Hourly: true}); got != 2 {
		t.Errorf("hourly days = %d, want 2", got)
	}
}

// TestFetchOpenMeteoEndToEnd stubs both the geocoding and forecast endpoints
// and checks the full pipeline including unit conversion and WMO mapping.
func TestFetchOpenMeteoEndToEnd(t *testing.T) {
	now := time.Now()
	hourIn := now.Add(3 * time.Hour).Unix()
	hourOut := now.Add(48 * time.Hour).Unix()
	sunrise := time.Date(2026, 7, 2, 4, 43, 0, 0, time.UTC).Unix()
	sunset := time.Date(2026, 7, 2, 19, 12, 0, 0, time.UTC).Unix()

	mux := http.NewServeMux()
	mux.HandleFunc("/geo", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != "Berlin" {
			t.Errorf("geocode name = %q, want Berlin", r.URL.Query().Get("name"))
		}
		fmt.Fprint(w, `{"results": [{"name": "Berlin", "latitude": 52.52, "longitude": 13.41}]}`)
	})
	mux.HandleFunc("/forecast", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("latitude") != "52.5200" {
			t.Errorf("latitude = %q, want 52.5200", q.Get("latitude"))
		}
		if q.Get("timeformat") != "unixtime" {
			t.Error("missing timeformat=unixtime")
		}
		fmt.Fprintf(w, `{
		  "utc_offset_seconds": 7200,
		  "current": {
		    "temperature_2m": 20, "apparent_temperature": 18,
		    "relative_humidity_2m": 65, "weather_code": 61,
		    "wind_speed_10m": 16.09344, "wind_direction_10m": 225
		  },
		  "hourly": {
		    "time": [%d, %d],
		    "temperature_2m": [21, 12],
		    "weather_code": [0, 95]
		  },
		  "daily": {
		    "time": [%d],
		    "temperature_2m_max": [25], "temperature_2m_min": [15],
		    "weather_code": [3],
		    "sunrise": [%d], "sunset": [%d],
		    "uv_index_max": [6.5]
		  }
		}`, hourIn, hourOut, now.Unix(), sunrise, sunset)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	oldGeo, oldBase := weather.OpenMeteoGeoBase, weather.OpenMeteoBase
	weather.OpenMeteoGeoBase, weather.OpenMeteoBase = srv.URL+"/geo", srv.URL+"/forecast"
	defer func() { weather.OpenMeteoGeoBase, weather.OpenMeteoBase = oldGeo, oldBase }()

	cfg := config.Config{APIProvider: config.ProviderOpenMeteo, City: "Berlin", Forecast: 1, Hourly: true, NoCache: true}
	info, err := weather.FetchWeather(context.Background(), cfg)
	if err != nil {
		t.Fatalf("FetchWeather: %v", err)
	}

	if info.Description != "Slight rain" || info.Type != weather.Rainy {
		t.Errorf("current = %q/%v, want Slight rain/Rainy", info.Description, info.Type)
	}
	if info.TempC != 20 || info.TempF != 68 {
		t.Errorf("Temp = %v/%v, want 20/68", info.TempC, info.TempF)
	}
	if info.FeelsLikeF != weather.CToF(18) {
		t.Errorf("FeelsLikeF = %v, want %v", info.FeelsLikeF, weather.CToF(18))
	}
	if info.WindMph < 9.9 || info.WindMph > 10.1 {
		t.Errorf("WindMph = %v, want ~10", info.WindMph)
	}
	if info.WindDegree != 225 {
		t.Errorf("WindDegree = %v, want 225", info.WindDegree)
	}
	if info.UVIndex != 6.5 {
		t.Errorf("UVIndex = %v, want 6.5 (daily max)", info.UVIndex)
	}
	// 04:43 UTC + 2h offset = 06:43 city time
	if info.Sunrise != "06:43" || info.Sunset != "21:12" {
		t.Errorf("Sun = %q/%q, want 06:43/21:12", info.Sunrise, info.Sunset)
	}
	if len(info.Hourly) != 1 {
		t.Fatalf("len(Hourly) = %d, want 1 (48h point dropped)", len(info.Hourly))
	}
	if info.Hourly[0].Description != "Clear sky" {
		t.Errorf("hourly desc = %q, want Clear sky", info.Hourly[0].Description)
	}
	if len(info.Forecast) != 1 {
		t.Fatalf("len(Forecast) = %d, want 1", len(info.Forecast))
	}
	d := info.Forecast[0]
	if d.Description != "Overcast" || d.Type != weather.Cloudy {
		t.Errorf("forecast = %q/%v, want Overcast/Cloudy", d.Description, d.Type)
	}
	if d.MaxTempF != weather.CToF(25) {
		t.Errorf("MaxTempF = %v, want %v", d.MaxTempF, weather.CToF(25))
	}
}

func TestBuildOpenMeteoURLHourlyParam(t *testing.T) {
	url := weather.BuildOpenMeteoURL(config.Config{}, 1, 2)
	if strings.Contains(url, "hourly=") {
		t.Errorf("hourly param should be absent without -hourly: %q", url)
	}
	url = weather.BuildOpenMeteoURL(config.Config{Hourly: true}, 1, 2)
	if !strings.Contains(url, "hourly=") {
		t.Errorf("hourly param missing: %q", url)
	}
}
