package main

import (
	"fmt"
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

type WeatherType int

const (
	Unknown WeatherType = iota
	Sunny
	Cloudy
	Rainy
	Snowy
	Stormy
	Foggy
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
	return string(runes[:maxRunes-3]) + "..."
}

func padRight(s string, w int) string {
	if len([]rune(s)) >= w {
		return truncate(s, w)
	}
	return s + strings.Repeat(" ", w-len([]rune(s)))
}

func formatForecastDate(dateStr string) string {
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("Mon, Jan 2")
}

func Display(info WeatherInfo, config Config) {

	weatherType := ClassifyWeather(info.Description)
	color := ""
	emoji := ""
	reset := ""
	if config.Fancy {
		color = WeatherColor(weatherType)
		emoji = WeatherEmoji(weatherType) + " "
		reset = Reset
	}

	if len(info.Forecast) > 0 {

		top := "┌" + strings.Repeat("─", forecastDayW) + "┬" + strings.Repeat("─", forecastDateW) + "┬" + strings.Repeat("─", forecastCondW) + "┬" + strings.Repeat("─", forecastTempW) + "┬" + strings.Repeat("─", forecastTempW) + "┐"
		mid := "├" + strings.Repeat("─", forecastDayW) + "┼" + strings.Repeat("─", forecastDateW) + "┼" + strings.Repeat("─", forecastCondW) + "┼" + strings.Repeat("─", forecastTempW) + "┼" + strings.Repeat("─", forecastTempW) + "┤"
		bot := "└" + strings.Repeat("─", forecastDayW) + "┴" + strings.Repeat("─", forecastDateW) + "┴" + strings.Repeat("─", forecastCondW) + "┴" + strings.Repeat("─", forecastTempW) + "┴" + strings.Repeat("─", forecastTempW) + "┘"

		if config.Fancy {
			fmt.Println()
			fmt.Println(Bold + "  📅 Forecast" + Reset)
			fmt.Println("  " + top)
			fmt.Printf("  │%s│%s│%s│%s│%s│\n",
				padRight(" Day", forecastDayW),
				padRight(" Date", forecastDateW),
				padRight(" Conditions", forecastCondW),
				padRight(" Min", forecastTempW),
				padRight(" Max", forecastTempW))
			fmt.Println("  " + mid)
		} else {
			fmt.Println()
			fmt.Println("Forecast")
			fmt.Println(top)
			fmt.Printf("│%s│%s│%s│%s│%s│\n",
				padRight(" Day", forecastDayW),
				padRight(" Date", forecastDateW),
				padRight(" Conditions", forecastCondW),
				padRight(" Min", forecastTempW),
				padRight(" Max", forecastTempW))
			fmt.Println(mid)
		}

		for i := 0; i < len(info.Forecast) && i < config.Forecast; i++ {
			w := info.Forecast[i]
			date, _ := w["date"].(string)
			dateFormatted := formatForecastDate(date)
			maxtempc, _ := w["maxtempC"].(string)
			maxtempf, _ := w["maxtempF"].(string)
			mintempc, _ := w["mintempC"].(string)
			mintempf, _ := w["mintempF"].(string)

			var desc string
			if weatherArr, ok := w["hourly"].([]interface{}); ok && len(weatherArr) > 0 {
				if hour0, ok := weatherArr[0].(map[string]interface{}); ok {
					if descArr, ok := hour0["weatherDesc"].([]interface{}); ok && len(descArr) > 0 {
						if descMap, ok := descArr[0].(map[string]interface{}); ok {
							desc, _ = descMap["value"].(string)
						}
					}
				}
			} else if descArr, ok := w["weatherDesc"].([]map[string]interface{}); ok && len(descArr) > 0 {
				desc, _ = descArr[0]["value"].(string)
			}

			wt := ClassifyWeather(desc)
			rowColor := ""
			if config.Fancy {
				rowColor = WeatherColor(wt)
			}
			var maxTemp, minTemp string
			if config.Unit == "imperial" {
				maxTemp = maxtempf + "°F"
				minTemp = mintempf + "°F"
			} else {
				maxTemp = maxtempc + "°C"
				minTemp = mintempc + "°C"
			}

			dayLabel := fmt.Sprintf(" Day %d", i+1)
			dateCell := padRight(" "+dateFormatted, forecastDateW)
			minCell := padRight(" "+minTemp, forecastTempW)
			maxCell := padRight(" "+maxTemp, forecastTempW)
			condCell := padRight(truncate(desc, forecastCondW), forecastCondW)

			if config.Fancy {
				fmt.Printf("  │%s%s%s│%s│%s%s%s│%s│%s│\n",
					rowColor, padRight(dayLabel, forecastDayW), Reset,
					dateCell,
					rowColor, condCell, Reset,
					minCell, maxCell)
			} else {
				fmt.Printf("│%s│%s│%s│%s│%s│\n",
					padRight(dayLabel, forecastDayW), dateCell, condCell, minCell, maxCell)
			}
		}

		if config.Fancy {
			fmt.Println("  " + bot)
			fmt.Println()
		} else {
			fmt.Println(bot)
			fmt.Println()
		}
	} else {
		fmt.Printf("%s%sWeather: %s, %s, UVIndex %s%s\n", color, emoji, info.Temperature, info.Description, info.UVIndex, reset)
	}
}

func ClassifyWeather(desc string) WeatherType {
	desc = strings.ToLower(desc)
	switch {
	case strings.Contains(desc, "sun"), strings.Contains(desc, "clear"):
		return Sunny
	case strings.Contains(desc, "cloud"), strings.Contains(desc, "overcast"):
		return Cloudy
	case strings.Contains(desc, "rain"), strings.Contains(desc, "shower"):
		return Rainy
	case strings.Contains(desc, "snow"):
		return Snowy
	case strings.Contains(desc, "storm"), strings.Contains(desc, "thunder"):
		return Stormy
	case strings.Contains(desc, "fog"), strings.Contains(desc, "mist"):
		return Foggy
	default:
		return Unknown
	}
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
