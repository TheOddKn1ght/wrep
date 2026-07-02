package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
)

var weatherAPIBase = "https://api.weatherapi.com/v1"

type weatherAPIResponse struct {
	Current struct {
		TempC      float64 `json:"temp_c"`
		TempF      float64 `json:"temp_f"`
		FeelsLikeC float64 `json:"feelslike_c"`
		FeelsLikeF float64 `json:"feelslike_f"`
		Humidity   float64 `json:"humidity"`
		WindKph    float64 `json:"wind_kph"`
		WindMph    float64 `json:"wind_mph"`
		WindDegree float64 `json:"wind_degree"`
		UVIndex    float64 `json:"uv"`
		Condition  struct {
			Text string `json:"text"`
		} `json:"condition"`
	} `json:"current"`
	Forecast struct {
		ForecastDay []struct {
			Date string `json:"date"`
			Day  struct {
				MaxTempC  float64 `json:"maxtemp_c"`
				MaxTempF  float64 `json:"maxtemp_f"`
				MinTempC  float64 `json:"mintemp_c"`
				MinTempF  float64 `json:"mintemp_f"`
				Condition struct {
					Text string `json:"text"`
				} `json:"condition"`
			} `json:"day"`
			Astro struct {
				Sunrise string `json:"sunrise"`
				Sunset  string `json:"sunset"`
			} `json:"astro"`
			Hour []struct {
				TimeEpoch int64   `json:"time_epoch"`
				Time      string  `json:"time"`
				TempC     float64 `json:"temp_c"`
				TempF     float64 `json:"temp_f"`
				Condition struct {
					Text string `json:"text"`
				} `json:"condition"`
			} `json:"hour"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

func fetchWeatherAPI(ctx context.Context, cfg config.Config) (WeatherInfo, error) {
	body, err := GetJSON(ctx, cfg, BuildWeatherAPIURL(cfg))
	if err != nil {
		return WeatherInfo{}, err
	}
	return ParseWeatherAPI(body, cfg)
}

// weatherAPIDays returns how many forecast days to request: hourly mode needs
// at least 2 days of hour data to cover the next 24h.
func weatherAPIDays(cfg config.Config) int {
	days := cfg.Forecast
	if cfg.Hourly && days < 2 {
		days = 2
	}
	return days
}

func BuildWeatherAPIURL(cfg config.Config) string {
	days := weatherAPIDays(cfg)
	endpoint := weatherAPIBase + "/current.json"
	if days > 0 {
		endpoint = weatherAPIBase + "/forecast.json"
	}
	q := url.Values{}
	q.Set("key", cfg.APIKey)
	q.Set("q", cfg.City)
	if days > 0 {
		q.Set("days", strconv.Itoa(days))
	}
	return endpoint + "?" + q.Encode()
}

func ParseWeatherAPI(body []byte, cfg config.Config) (WeatherInfo, error) {
	var r weatherAPIResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	info := WeatherInfo{
		Description: strings.TrimSpace(r.Current.Condition.Text),
		TempC:       r.Current.TempC,
		TempF:       r.Current.TempF,
		FeelsLikeC:  r.Current.FeelsLikeC,
		FeelsLikeF:  r.Current.FeelsLikeF,
		Humidity:    r.Current.Humidity,
		WindKph:     r.Current.WindKph,
		WindMph:     r.Current.WindMph,
		WindDegree:  r.Current.WindDegree,
		UVIndex:     r.Current.UVIndex,
	}
	info.Type = ClassifyWeather(info.Description)

	if len(r.Forecast.ForecastDay) > 0 {
		info.Sunrise = r.Forecast.ForecastDay[0].Astro.Sunrise
		info.Sunset = r.Forecast.ForecastDay[0].Astro.Sunset
	}

	if cfg.Hourly {
		now := time.Now()
		for _, day := range r.Forecast.ForecastDay {
			for _, h := range day.Hour {
				t := time.Unix(h.TimeEpoch, 0)
				if t.Before(now.Add(-90*time.Minute)) || t.After(now.Add(24*time.Hour)) {
					continue
				}
				// Display the city-local wall clock time from the "time"
				// string; the epoch is only used for windowing.
				label, err := time.Parse("2006-01-02 15:04", h.Time)
				if err != nil {
					label = t
				}
				desc := strings.TrimSpace(h.Condition.Text)
				info.Hourly = append(info.Hourly, HourlyPoint{
					Time:        label,
					TempC:       h.TempC,
					TempF:       h.TempF,
					Description: desc,
					Type:        ClassifyWeather(desc),
				})
			}
		}
	}

	if cfg.Forecast > 0 {
		for _, day := range r.Forecast.ForecastDay {
			d, _ := time.Parse("2006-01-02", day.Date)
			desc := strings.TrimSpace(day.Day.Condition.Text)
			info.Forecast = append(info.Forecast, ForecastDay{
				Date:        d,
				MaxTempC:    day.Day.MaxTempC,
				MinTempC:    day.Day.MinTempC,
				MaxTempF:    day.Day.MaxTempF,
				MinTempF:    day.Day.MinTempF,
				Description: desc,
				Type:        ClassifyWeather(desc),
			})
		}
	}
	return info, nil
}
