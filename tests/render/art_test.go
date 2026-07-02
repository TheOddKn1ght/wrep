package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/render"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

func TestWeatherArtCoversAllTypes(t *testing.T) {
	types := []weather.WeatherType{
		weather.Unknown, weather.Sunny, weather.Cloudy, weather.Rainy,
		weather.Snowy, weather.Stormy, weather.Foggy,
	}
	for _, wt := range types {
		if len(render.WeatherArt(wt)) == 0 {
			t.Errorf("WeatherArt(%v) is empty", wt)
		}
	}
	if got := render.WeatherArt(weather.WeatherType(99)); len(got) == 0 {
		t.Error("WeatherArt should fall back to Unknown art for unmapped types")
	}
}

func TestRenderArt(t *testing.T) {
	var buf bytes.Buffer
	info := weather.WeatherInfo{
		TempC:       21,
		TempF:       70,
		FeelsLikeC:  19,
		FeelsLikeF:  66,
		Humidity:    40,
		WindKph:     12,
		WindMph:     7,
		WindDegree:  315,
		UVIndex:     4,
		Sunrise:     "06:43 AM",
		Sunset:      "09:12 PM",
		Description: "Sunny",
		Type:        weather.Sunny,
	}
	render.Display(&buf, info, config.Config{Art: true, City: "Berlin", Unit: config.UnitMetric})
	out := buf.String()

	for _, want := range []string{"City", "Berlin", "Condition", "Sunny", "Temperature", "21.0°C", "Feels like", "Humidity", "40%", "Wind", "12 km/h NW", "UV Index", "4.0", "Sunrise", "06:43 AM", "Sunset", "09:12 PM"} {
		if !strings.Contains(out, want) {
			t.Errorf("art output missing %q:\n%s", want, out)
		}
	}
	// every info line must be present even when the art has fewer rows
	if got := strings.Count(out, ": "); got < 9 {
		t.Errorf("expected at least 9 info lines, found %d:\n%s", got, out)
	}
}

func TestRenderArtNoAstro(t *testing.T) {
	var buf bytes.Buffer
	info := weather.WeatherInfo{Description: "Sunny", Type: weather.Sunny}
	render.Display(&buf, info, config.Config{Art: true, City: "Berlin", Unit: config.UnitMetric})
	if strings.Contains(buf.String(), "Sunrise") {
		t.Errorf("Sunrise row should be hidden without astro data:\n%s", buf.String())
	}
}
