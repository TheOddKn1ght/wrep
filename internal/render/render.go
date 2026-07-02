// Package render turns normalized weather data into terminal output: plain
// text, Unicode tables, sparklines, ASCII art, JSON, and custom templates.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Gray    = "\033[90m"
	Bold    = "\033[1m"
)

const (
	forecastDayW  = 7
	forecastDateW = 12
	forecastCondW = 24
	forecastTempW = 8
)

func Truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func PadRight(s string, w int) string {
	if len([]rune(s)) >= w {
		return Truncate(s, w)
	}
	return s + strings.Repeat(" ", w-len([]rune(s)))
}

func Display(w io.Writer, info weather.WeatherInfo, cfg config.Config) {
	switch {
	case cfg.JSON:
		renderJSON(w, info)
	case cfg.Format != "":
		renderFormat(w, info, cfg)
	case cfg.Art:
		renderArt(w, info, cfg)
	case cfg.Hourly:
		renderCurrent(w, info, cfg)
		renderHourly(w, info, cfg)
		if len(info.Forecast) > 0 {
			renderForecast(w, info, cfg)
		}
	case len(info.Forecast) > 0:
		renderForecast(w, info, cfg)
	default:
		renderCurrent(w, info, cfg)
	}
}

func renderJSON(w io.Writer, info weather.WeatherInfo) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(info)
}

func renderCurrent(w io.Writer, info weather.WeatherInfo, cfg config.Config) {
	color, emoji, reset := "", "", ""
	if useColor(cfg) {
		color = WeatherColor(info.Type)
		reset = Reset
	}
	if cfg.Fancy {
		emoji = WeatherEmoji(info.Type) + " "
	}
	fmt.Fprintf(w, "%s%sWeather: %s (feels like %s), %s, humidity %.0f%%, wind %s %s, UVIndex %s%s\n",
		color, emoji,
		FormatTemp(info.TempC, info.TempF, cfg.Unit),
		FormatTemp(info.FeelsLikeC, info.FeelsLikeF, cfg.Unit),
		info.Description,
		info.Humidity,
		FormatWind(info.WindKph, info.WindMph, cfg.Unit),
		weather.WindCompass(info.WindDegree),
		formatUV(info.UVIndex),
		reset,
	)
}

func renderForecast(w io.Writer, info weather.WeatherInfo, cfg config.Config) {
	top, mid, bot := forecastBorders()
	indent := ""
	color := useColor(cfg)
	if cfg.Fancy {
		indent = "  "
		fmt.Fprintln(w)
		header := "  📅 Forecast"
		if color {
			header = "  " + Bold + "📅 Forecast" + Reset
		}
		fmt.Fprintln(w, header)
	} else {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Forecast")
	}

	fmt.Fprintln(w, indent+top)
	fmt.Fprintf(w, indent+"│%s│%s│%s│%s│%s│\n",
		PadRight(" Day", forecastDayW),
		PadRight(" Date", forecastDateW),
		PadRight(" Conditions", forecastCondW),
		PadRight(" Min", forecastTempW),
		PadRight(" Max", forecastTempW))
	fmt.Fprintln(w, indent+mid)

	limit := len(info.Forecast)
	if cfg.Forecast > 0 && cfg.Forecast < limit {
		limit = cfg.Forecast
	}
	for i := 0; i < limit; i++ {
		d := info.Forecast[i]
		dayLabel := fmt.Sprintf(" Day %d", i+1)
		dateCell := PadRight(" "+formatForecastDate(d.Date), forecastDateW)
		condCell := PadRight(" "+Truncate(d.Description, forecastCondW-1), forecastCondW)
		minCell := PadRight(" "+FormatTemp(d.MinTempC, d.MinTempF, cfg.Unit), forecastTempW)
		maxCell := PadRight(" "+FormatTemp(d.MaxTempC, d.MaxTempF, cfg.Unit), forecastTempW)

		if color {
			rowColor := WeatherColor(d.Type)
			fmt.Fprintf(w, indent+"│%s%s%s│%s│%s%s%s│%s│%s│\n",
				rowColor, PadRight(dayLabel, forecastDayW), Reset,
				dateCell,
				rowColor, condCell, Reset,
				minCell, maxCell)
		} else {
			fmt.Fprintf(w, indent+"│%s│%s│%s│%s│%s│\n",
				PadRight(dayLabel, forecastDayW), dateCell, condCell, minCell, maxCell)
		}
	}

	fmt.Fprintln(w, indent+bot)
	fmt.Fprintln(w)
}

func formatForecastDate(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("Mon, Jan 2")
}

func FormatTemp(c, f float64, unit string) string {
	if unit == config.UnitImperial {
		return fmt.Sprintf("%.1f°F", f)
	}
	return fmt.Sprintf("%.1f°C", c)
}

func formatUV(uv float64) string {
	return fmt.Sprintf("%.1f", uv)
}

func FormatWind(kph, mph float64, unit string) string {
	if unit == config.UnitImperial {
		return fmt.Sprintf("%.0f mph", mph)
	}
	return fmt.Sprintf("%.0f km/h", kph)
}

func forecastBorders() (top, mid, bot string) {
	top = "┌" + strings.Repeat("─", forecastDayW) + "┬" + strings.Repeat("─", forecastDateW) + "┬" + strings.Repeat("─", forecastCondW) + "┬" + strings.Repeat("─", forecastTempW) + "┬" + strings.Repeat("─", forecastTempW) + "┐"
	mid = "├" + strings.Repeat("─", forecastDayW) + "┼" + strings.Repeat("─", forecastDateW) + "┼" + strings.Repeat("─", forecastCondW) + "┼" + strings.Repeat("─", forecastTempW) + "┼" + strings.Repeat("─", forecastTempW) + "┤"
	bot = "└" + strings.Repeat("─", forecastDayW) + "┴" + strings.Repeat("─", forecastDateW) + "┴" + strings.Repeat("─", forecastCondW) + "┴" + strings.Repeat("─", forecastTempW) + "┴" + strings.Repeat("─", forecastTempW) + "┘"
	return
}

func useColor(cfg config.Config) bool {
	if !cfg.Fancy {
		return false
	}
	if cfg.NoColor {
		return false
	}
	if v := os.Getenv("NO_COLOR"); v != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return StdoutIsTTY()
}

// StdoutIsTTY reports whether stdout is attached to a terminal; used both for
// color decisions here and screen-clearing in live mode.
func StdoutIsTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return true
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func WeatherColor(wt weather.WeatherType) string {
	switch wt {
	case weather.Sunny:
		return Yellow
	case weather.Cloudy:
		return White
	case weather.Rainy:
		return Blue
	case weather.Snowy:
		return Cyan
	case weather.Stormy:
		return Magenta
	case weather.Foggy:
		return Gray
	default:
		return Green
	}
}

func WeatherEmoji(wt weather.WeatherType) string {
	switch wt {
	case weather.Sunny:
		return "☀️"
	case weather.Cloudy:
		return "☁️"
	case weather.Rainy:
		return "🌧️"
	case weather.Snowy:
		return "❄️"
	case weather.Stormy:
		return "⛈️"
	case weather.Foggy:
		return "🌫️"
	default:
		return "🌈"
	}
}
