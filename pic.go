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
)

func Display(info WeatherInfo, fancy bool) {
	if fancy {
		fmt.Println(getWeatherPicture(info.Description))
		//fmt.Println("[debug] description (lowercased):", strings.ToLower(info.Description))
	}
	fmt.Printf(Cyan+"Weather: %s, %s, UVIndex %s\n"+Reset, info.Temperature, info.Description, info.UVIndex)
}

func getWeatherPicture(desc string) string {
	desc = strings.ToLower(desc)
	switch {
	case strings.Contains(desc, "sun") || strings.Contains(desc, "clear"):
		return "☀️  Sunny"
	case strings.Contains(desc, "cloudy"):
		return "☁️  Cloudy"
	case strings.Contains(desc, "rain"):
		return "🌧️  Rainy"
	case strings.Contains(desc, "snow"):
		return "❄️  Snowy"
	case strings.Contains(desc, "storm") || strings.Contains(desc, "thunder"):
		return "⛈️  Stormy"
	case strings.Contains(desc, "fog") || strings.Contains(desc, "mist"):
		return "🌫️  Foggy"
	default:
		return "🌈  Weather Unknown"
	}
}
