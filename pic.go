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


func Display(info WeatherInfo, fancy bool) {
	weatherType := ClassifyWeather(info.Description)

	color := ""
	emoji := ""
	reset := ""
	if fancy {
		color = weatherColor(weatherType)
		emoji = weatherEmoji(weatherType) + " "
		reset = Reset
	}

	fmt.Printf("%s%sWeather: %s, %s, UVIndex %s%s\n", color, emoji, info.Temperature, info.Description, info.UVIndex, reset)
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

func weatherColor(wt WeatherType) string {
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

func weatherEmoji(wt WeatherType) string {
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
