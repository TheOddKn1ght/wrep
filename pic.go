package main

import (
	"fmt"
	"strings"
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
		if config.Fancy {
			fmt.Println(Bold + "Forecast:" + Reset)
		} else {
			fmt.Println("Forecast:")
		}

		for i := 0; i < len(info.Forecast) && i < config.Forecast; i++ {
			w := info.Forecast[i]
			date, _ := w["date"].(string)
			maxtempc, _ := w["maxtempC"].(string)
			maxtempf, _ := w["maxtempF"].(string)
			mintempc, _ := w["mintempC"].(string)
			mintempf, _ := w["mintempF"].(string)
			weatherArr, _ := w["hourly"].([]interface{})
			var desc string
			if len(weatherArr) > 0 {
				hour0, _ := weatherArr[0].(map[string]interface{})
				if hour0 != nil {
					if descArr, ok := hour0["weatherDesc"].([]interface{}); ok && len(descArr) > 0 {
						descMap, _ := descArr[0].(map[string]interface{})
						if descMap != nil {
							desc, _ = descMap["value"].(string)
						}
					}
				}
			}

			wt := ClassifyWeather(desc)
			color := ""
			emoji := ""
			reset := ""
			if config.Fancy {
				color = WeatherColor(wt)
				emoji = WeatherEmoji(wt) + " "
				reset = Reset
			}
			var maxTemp, minTemp string
			if config.Unit == "imperial" {
				maxTemp = maxtempf + "°F"
				minTemp = mintempf + "°F"
			} else {
				maxTemp = maxtempc + "°C"
				minTemp = mintempc + "°C"
			}
			fmt.Printf("%s%sDay %d (%s): %s, %s - %s%s\n", color, emoji, i+1, date, desc, minTemp, maxTemp, reset)
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
