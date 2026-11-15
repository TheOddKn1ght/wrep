package render_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/render"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

func TestSparkline(t *testing.T) {
	got := render.Sparkline([]float64{0, 1, 2, 3, 4, 5, 6, 7})
	if got != "▁▂▃▄▅▆▇█" {
		t.Errorf("Sparkline ramp = %q, want ▁▂▃▄▅▆▇█", got)
	}

	got = render.Sparkline([]float64{5, 5, 5})
	if got != "▅▅▅" {
		t.Errorf("flat Sparkline = %q, want mid-height ▅▅▅", got)
	}

	if got := render.Sparkline(nil); got != "" {
		t.Errorf("empty Sparkline = %q, want empty", got)
	}

	got = render.Sparkline([]float64{10, 20, 10})
	if []rune(got)[0] != '▁' || []rune(got)[1] != '█' {
		t.Errorf("min/max mapping wrong: %q", got)
	}
}

func hourlyTestInfo(n int) weather.WeatherInfo {
	info := weather.WeatherInfo{Description: "Sunny", Type: weather.Sunny, TempC: 20, TempF: 68}
	base := time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC)
	for i := 0; i < n; i++ {
		c := 15 + float64(i%10)
		info.Hourly = append(info.Hourly, weather.HourlyPoint{
			Time:        base.Add(time.Duration(i) * time.Hour),
			TempC:       c,
			TempF:       c*9/5 + 32,
			Description: "Sunny",
			Type:        weather.Sunny,
		})
	}
	return info
}

func TestRenderHourly(t *testing.T) {
	var buf bytes.Buffer
	render.Display(&buf, hourlyTestInfo(8), config.Config{Hourly: true, Unit: config.UnitMetric})
	out := buf.String()

	if !strings.Contains(out, "Next 24 hours") {
		t.Errorf("missing header:\n%s", out)
	}
	if !strings.Contains(out, "12:00") {
		t.Errorf("missing first time slot:\n%s", out)
	}
	// current conditions line is shown above the hourly block
	if !strings.Contains(out, "Weather: 20.0°C") {
		t.Errorf("missing current conditions:\n%s", out)
	}
	// min–max range line
	if !strings.Contains(out, "15.0°C – 22.0°C") {
		t.Errorf("missing temp range:\n%s", out)
	}
	if !strings.ContainsAny(out, "▁▂▃▄▅▆▇█") {
		t.Errorf("missing sparkline:\n%s", out)
	}
}

func TestRenderHourlySampling(t *testing.T) {
	var buf bytes.Buffer
	render.Display(&buf, hourlyTestInfo(25), config.Config{Hourly: true, Unit: config.UnitMetric})
	out := buf.String()

	rows := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "°C") && strings.Contains(line, ":00") && !strings.Contains(line, "–") {
			rows++
		}
	}
	if rows > 9 {
		t.Errorf("hourly rows = %d, want at most 9 (sampled):\n%s", rows, out)
	}
	// the sparkline still uses every point
	sparkLen := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.ContainsAny(line, "▁▂▃▄▅▆▇█") {
			sparkLen = strings.Count(line, "▁") + strings.Count(line, "▂") + strings.Count(line, "▃") +
				strings.Count(line, "▄") + strings.Count(line, "▅") + strings.Count(line, "▆") +
				strings.Count(line, "▇") + strings.Count(line, "█")
		}
	}
	if sparkLen != 25 {
		t.Errorf("sparkline length = %d, want 25 (all points)", sparkLen)
	}
}

func TestRenderHourlyEmpty(t *testing.T) {
	var buf bytes.Buffer
	info := hourlyTestInfo(0)
	render.Display(&buf, info, config.Config{Hourly: true, Quiet: true, Unit: config.UnitMetric})
	// must not panic and must still show current conditions
	if !strings.Contains(buf.String(), "Weather:") {
		t.Errorf("expected current conditions, got:\n%s", buf.String())
	}
}

func TestRenderHourlyWithForecast(t *testing.T) {
	info := hourlyTestInfo(8)
	info.Forecast = testInfo().Forecast
	var buf bytes.Buffer
	render.Display(&buf, info, config.Config{Hourly: true, Forecast: 2, Unit: config.UnitMetric})
	out := buf.String()
	if !strings.Contains(out, "Next 24 hours") || !strings.Contains(out, "Day 1") {
		t.Errorf("combined hourly+forecast view missing a section:\n%s", out)
	}
}
