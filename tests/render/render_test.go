package render_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/render"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		s    string
		max  int
		want string
	}{
		{"short", 10, "short"},
		{"exactly-ten", 11, "exactly-ten"},
		{"this is far too long", 10, "this is..."},
		{"abc", 2, "ab"},
		{"héllo wörld", 8, "héllo..."},
	}
	for _, tt := range tests {
		if got := render.Truncate(tt.s, tt.max); got != tt.want {
			t.Errorf("Truncate(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
		}
	}
}

func TestPadRight(t *testing.T) {
	if got := render.PadRight("ab", 5); got != "ab   " {
		t.Errorf("PadRight(\"ab\", 5) = %q", got)
	}
	if got := render.PadRight("abcdef", 4); len([]rune(got)) != 4 {
		t.Errorf("PadRight should truncate to width, got %q", got)
	}
	if got := render.PadRight("héllo", 6); got != "héllo " {
		t.Errorf("PadRight should count runes, got %q", got)
	}
}

func TestFormatTemp(t *testing.T) {
	if got := render.FormatTemp(21.5, 70.7, config.UnitMetric); got != "21.5°C" {
		t.Errorf("metric = %q", got)
	}
	if got := render.FormatTemp(21.5, 70.7, config.UnitImperial); got != "70.7°F" {
		t.Errorf("imperial = %q", got)
	}
}

func TestFormatWind(t *testing.T) {
	if got := render.FormatWind(14.4, 8.9, config.UnitMetric); got != "14 km/h" {
		t.Errorf("metric = %q", got)
	}
	if got := render.FormatWind(14.4, 8.9, config.UnitImperial); got != "9 mph" {
		t.Errorf("imperial = %q", got)
	}
}

func testInfo() weather.WeatherInfo {
	return weather.WeatherInfo{
		TempC:       21,
		TempF:       70,
		FeelsLikeC:  19,
		FeelsLikeF:  66,
		Humidity:    40,
		WindKph:     12,
		WindMph:     7,
		WindDegree:  315,
		UVIndex:     4,
		Description: "Partly cloudy",
		Type:        weather.Cloudy,
		Forecast: []weather.ForecastDay{
			{
				Date:        time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
				MinTempC:    15,
				MaxTempC:    25,
				MinTempF:    59,
				MaxTempF:    77,
				Description: "Light rain shower with extras that overflow",
				Type:        weather.Rainy,
			},
			{
				Date:        time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
				MinTempC:    16,
				MaxTempC:    26,
				MinTempF:    61,
				MaxTempF:    79,
				Description: "Sunny",
				Type:        weather.Sunny,
			},
		},
	}
}

func TestRenderCurrentPlain(t *testing.T) {
	var buf bytes.Buffer
	info := testInfo()
	info.Forecast = nil
	render.Display(&buf, info, config.Config{Unit: config.UnitMetric})
	got := buf.String()
	want := "Weather: 21.0°C (feels like 19.0°C), Partly cloudy, humidity 40%, wind 12 km/h NW, UVIndex 4.0\n"
	if got != want {
		t.Errorf("renderCurrent:\n got %q\nwant %q", got, want)
	}
}

func TestRenderForecastAlignment(t *testing.T) {
	var buf bytes.Buffer
	render.Display(&buf, testInfo(), config.Config{Unit: config.UnitMetric, Forecast: 2})
	out := buf.String()

	if !strings.Contains(out, "Forecast") {
		t.Fatalf("missing Forecast header in output:\n%s", out)
	}

	var width int
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "┌") && !strings.HasPrefix(line, "│") &&
			!strings.HasPrefix(line, "├") && !strings.HasPrefix(line, "└") {
			continue
		}
		n := len([]rune(line))
		if width == 0 {
			width = n
		} else if n != width {
			t.Errorf("table line width %d differs from %d: %q", n, width, line)
		}
	}
	if width == 0 {
		t.Fatalf("no table lines found in output:\n%s", out)
	}

	if !strings.Contains(out, "Day 1") || !strings.Contains(out, "Day 2") {
		t.Errorf("missing day rows:\n%s", out)
	}
	if !strings.Contains(out, "...") {
		t.Errorf("long description should be truncated with ellipsis:\n%s", out)
	}
}

func TestRenderForecastLimit(t *testing.T) {
	var buf bytes.Buffer
	render.Display(&buf, testInfo(), config.Config{Unit: config.UnitMetric, Forecast: 1})
	out := buf.String()
	if strings.Contains(out, "Day 2") {
		t.Errorf("Forecast=1 should hide Day 2:\n%s", out)
	}
}

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	render.Display(&buf, testInfo(), config.Config{JSON: true, Forecast: 2})
	var decoded weather.WeatherInfo
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("JSON output not parseable: %v\n%s", err, buf.String())
	}
	if decoded.TempC != 21 || decoded.Humidity != 40 || decoded.WindKph != 12 {
		t.Errorf("JSON round-trip lost data: %+v", decoded)
	}
	if len(decoded.Forecast) != 2 {
		t.Errorf("JSON forecast length = %d, want 2", len(decoded.Forecast))
	}
}

func TestWeatherColorAndEmojiCoverAllTypes(t *testing.T) {
	types := []weather.WeatherType{
		weather.Unknown, weather.Sunny, weather.Cloudy, weather.Rainy,
		weather.Snowy, weather.Stormy, weather.Foggy,
	}
	for _, wt := range types {
		if render.WeatherColor(wt) == "" {
			t.Errorf("WeatherColor(%v) is empty", wt)
		}
		if render.WeatherEmoji(wt) == "" {
			t.Errorf("WeatherEmoji(%v) is empty", wt)
		}
	}
}
