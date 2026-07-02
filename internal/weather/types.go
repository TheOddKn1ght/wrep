// Package weather fetches and normalizes weather data from the supported
// providers (wttr.in, WeatherAPI, Open-Meteo) behind a disk cache with retry
// and stale-fallback behavior.
package weather

import (
	"math"
	"strings"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
)

// ForecastCaps holds the maximum forecast days each provider can return, for
// providers where exceeding it silently truncates.
var ForecastCaps = map[string]int{
	config.ProviderWttr:      3,
	config.ProviderOpenMeteo: 16,
}

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

type WeatherInfo struct {
	TempC       float64       `json:"temp_c"`
	TempF       float64       `json:"temp_f"`
	FeelsLikeC  float64       `json:"feelslike_c"`
	FeelsLikeF  float64       `json:"feelslike_f"`
	Humidity    float64       `json:"humidity"`
	WindKph     float64       `json:"wind_kph"`
	WindMph     float64       `json:"wind_mph"`
	WindDegree  float64       `json:"wind_degree"`
	UVIndex     float64       `json:"uv_index"`
	Sunrise     string        `json:"sunrise,omitempty"`
	Sunset      string        `json:"sunset,omitempty"`
	Description string        `json:"description"`
	Type        WeatherType   `json:"-"`
	Hourly      []HourlyPoint `json:"hourly,omitempty"`
	Forecast    []ForecastDay `json:"forecast,omitempty"`
}

type HourlyPoint struct {
	Time        time.Time   `json:"time"`
	TempC       float64     `json:"temp_c"`
	TempF       float64     `json:"temp_f"`
	Description string      `json:"description"`
	Type        WeatherType `json:"-"`
}

type ForecastDay struct {
	Date        time.Time   `json:"date"`
	MinTempC    float64     `json:"min_temp_c"`
	MaxTempC    float64     `json:"max_temp_c"`
	MinTempF    float64     `json:"min_temp_f"`
	MaxTempF    float64     `json:"max_temp_f"`
	Description string      `json:"description"`
	Type        WeatherType `json:"-"`
}

// ClassifyWeather maps a free-text condition to a WeatherType. Order matters:
// mixed descriptions like "rain with thunder" should classify as the more
// severe phenomenon.
func ClassifyWeather(desc string) WeatherType {
	desc = strings.ToLower(desc)
	has := func(subs ...string) bool {
		for _, s := range subs {
			if strings.Contains(desc, s) {
				return true
			}
		}
		return false
	}
	switch {
	case has("storm", "thunder"):
		return Stormy
	case has("snow", "sleet", "blizzard"):
		return Snowy
	case has("rain", "shower", "drizzle"):
		return Rainy
	case has("fog", "mist", "haze"):
		return Foggy
	case has("sun", "clear"):
		return Sunny
	case has("cloud", "overcast"):
		return Cloudy
	default:
		return Unknown
	}
}

var compassPoints = [16]string{
	"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
	"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW",
}

// WindCompass converts meteorological degrees (wind origin) to a 16-point
// compass label.
func WindCompass(deg float64) string {
	deg = math.Mod(deg, 360)
	if deg < 0 {
		deg += 360
	}
	return compassPoints[int(deg/22.5+0.5)%16]
}

func CToF(c float64) float64 {
	return c*9/5 + 32
}

func KphToMph(kph float64) float64 {
	return kph / 1.609344
}
