package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
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

func truncate(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func padRight(s string, w int) string {
	if len([]rune(s)) >= w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-len([]rune(s)))
}

func Display(w io.Writer, info WeatherInfo, config Config) {
	if config.JSON {
		renderJSON(w, info)
		return
	}
	if len(info.Forecast) > 0 {
		renderForecast(w, info, config)
		return
	}
	renderCurrent(w, info, config)
}

func renderJSON(w io.Writer, info WeatherInfo) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(info)
}

func renderCurrent(w io.Writer, info WeatherInfo, config Config) {
	color, emoji, reset := "", "", ""
	if useColor(config) {
		color = WeatherColor(info.Type)
		reset = Reset
	}
	if config.Fancy {
		emoji = WeatherEmoji(info.Type) + " "
	}
	fmt.Fprintf(w, "%s%sWeather: %s, %s, UVIndex %s%s\n",
		color, emoji,
		formatTemp(info.TempC, info.TempF, config.Unit),
		info.Description,
		formatUV(info.UVIndex),
		reset,
	)
}

func renderForecast(w io.Writer, info WeatherInfo, config Config) {
	top, mid, bot := forecastBorders()
	indent := ""
	color := useColor(config)
	if config.Fancy {
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
		padRight(" Day", forecastDayW),
		padRight(" Date", forecastDateW),
		padRight(" Conditions", forecastCondW),
		padRight(" Min", forecastTempW),
		padRight(" Max", forecastTempW))
	fmt.Fprintln(w, indent+mid)

	limit := len(info.Forecast)
	if config.Forecast > 0 && config.Forecast < limit {
		limit = config.Forecast
	}
	for i := 0; i < limit; i++ {
		d := info.Forecast[i]
		dayLabel := fmt.Sprintf(" Day %d", i+1)
		dateCell := padRight(" "+formatForecastDate(d.Date), forecastDateW)
		condCell := padRight(truncate(d.Description, forecastCondW), forecastCondW)
		minCell := padRight(" "+formatTemp(d.MinTempC, d.MinTempF, config.Unit), forecastTempW)
		maxCell := padRight(" "+formatTemp(d.MaxTempC, d.MaxTempF, config.Unit), forecastTempW)

		if color {
			rowColor := WeatherColor(d.Type)
			fmt.Fprintf(w, indent+"│%s%s%s│%s│%s%s%s│%s│%s│\n",
				rowColor, padRight(dayLabel, forecastDayW), Reset,
				dateCell,
				rowColor, condCell, Reset,
				minCell, maxCell)
		} else {
			fmt.Fprintf(w, indent+"│%s│%s│%s│%s│%s│\n",
				padRight(dayLabel, forecastDayW), dateCell, condCell, minCell, maxCell)
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

func formatTemp(c, f float64, unit string) string {
	if unit == UnitImperial {
		return fmt.Sprintf("%.1f°F", f)
	}
	return fmt.Sprintf("%.1f°C", c)
}

func formatUV(uv float64) string {
	return fmt.Sprintf("%.1f", uv)
}

func forecastBorders() (top, mid, bot string) {
	top = "┌" + strings.Repeat("─", forecastDayW) + "┬" + strings.Repeat("─", forecastDateW) + "┬" + strings.Repeat("─", forecastCondW) + "┬" + strings.Repeat("─", forecastTempW) + "┬" + strings.Repeat("─", forecastTempW) + "┐"
	mid = "├" + strings.Repeat("─", forecastDayW) + "┼" + strings.Repeat("─", forecastDateW) + "┼" + strings.Repeat("─", forecastCondW) + "┼" + strings.Repeat("─", forecastTempW) + "┼" + strings.Repeat("─", forecastTempW) + "┤"
	bot = "└" + strings.Repeat("─", forecastDayW) + "┴" + strings.Repeat("─", forecastDateW) + "┴" + strings.Repeat("─", forecastCondW) + "┴" + strings.Repeat("─", forecastTempW) + "┴" + strings.Repeat("─", forecastTempW) + "┘"
	return
}

func useColor(config Config) bool {
	if !config.Fancy {
		return false
	}
	if config.NoColor {
		return false
	}
	if v := os.Getenv("NO_COLOR"); v != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return true
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func WeatherColor(wt WeatherType) string {
	switch wt {
	case Sunny:
		return Yellow
	case Cloudy:
		return White
	case Rainy:
		return Blue
	case Snowy:
		return Cyan
	case Stormy:
		return Magenta
	case Foggy:
		return Gray
	default:
		return Green
	}
}

func WeatherEmoji(wt WeatherType) string {
	switch wt {
	case Sunny:
		return "☀️"
	case Cloudy:
		return "☁️"
	case Rainy:
		return "🌧️"
	case Snowy:
		return "❄️"
	case Stormy:
		return "⛈️"
	case Foggy:
		return "🌫️"
	default:
		return "🌈"
	}
}
