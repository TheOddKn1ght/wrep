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
            fmt.Println("-------------------------------------------------------------------------")
            fmt.Printf("%-10s %-15s %-24s %-10s %-10s\n", "Day", "Date", "Description", "Min Temp", "Max Temp")
            fmt.Println("-------------------------------------------------------------------------")
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

            if config.Fancy {
                fmt.Printf("%s%-10s %-15s %-25s %-10s %-10s%s\n",
                    color, fmt.Sprintf("Day %d", i+1), date, emoji+" "+desc, minTemp, maxTemp, reset)
            } else {
                fmt.Printf("Day %d (%s): %s, %s - %s\n", i+1, date, desc, minTemp, maxTemp)
            }
        }
        if config.Fancy {
            fmt.Println("-------------------------------------------------------------------------")
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
