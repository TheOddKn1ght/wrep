package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/render"
)

func TestExpandFormat(t *testing.T) {
	info := testInfo()
	cfg := config.Config{City: "Berlin", Unit: config.UnitMetric}

	tests := []struct {
		format string
		want   string
	}{
		{"{city}: {temp}", "Berlin: 21.0°C"},
		{"{emoji} {desc}", "☁️ Partly cloudy"},
		{"{feels}", "19.0°C"},
		{"{tempc}/{tempf}", "21.0/70.0"},
		{"humidity {humidity}%", "humidity 40%"},
		{"{wind} {winddir}", "12 km/h NW"},
		{"UV {uv}", "UV 4.0"},
		{"{min}..{max}", "15.0°C..25.0°C"},
		{"{unknown} stays", "{unknown} stays"},
		{"no placeholders", "no placeholders"},
	}
	for _, tt := range tests {
		if got := render.ExpandFormat(tt.format, info, cfg); got != tt.want {
			t.Errorf("ExpandFormat(%q) = %q, want %q", tt.format, got, tt.want)
		}
	}
}

func TestExpandFormatImperial(t *testing.T) {
	got := render.ExpandFormat("{temp} {wind}", testInfo(), config.Config{Unit: config.UnitImperial})
	if got != "70.0°F 7 mph" {
		t.Errorf("imperial format = %q", got)
	}
}

func TestExpandFormatNoForecast(t *testing.T) {
	info := testInfo()
	info.Forecast = nil
	got := render.ExpandFormat("[{min}][{max}]", info, config.Config{Unit: config.UnitMetric})
	if got != "[][]" {
		t.Errorf("min/max without forecast = %q, want empty substitutions", got)
	}
}

func TestRenderFormatViaDisplay(t *testing.T) {
	var buf bytes.Buffer
	render.Display(&buf, testInfo(), config.Config{Format: "{city} {temp}", City: "Oslo", Unit: config.UnitMetric})
	got := buf.String()
	if got != "Oslo 21.0°C\n" {
		t.Errorf("Display with format = %q", got)
	}
	if strings.Contains(got, "\033") {
		t.Error("format output must not contain ANSI escapes")
	}
}
