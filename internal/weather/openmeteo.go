package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
)

var (
	OpenMeteoGeoBase = "https://geocoding-api.open-meteo.com/v1/search"
	OpenMeteoBase    = "https://api.open-meteo.com/v1/forecast"
)

// WMODescriptions maps WMO weather interpretation codes (as returned by
// Open-Meteo) to human-readable conditions.
var WMODescriptions = map[int]string{
	0:  "Clear sky",
	1:  "Mainly clear",
	2:  "Partly cloudy",
	3:  "Overcast",
	45: "Fog",
	48: "Depositing rime fog",
	51: "Light drizzle",
	53: "Moderate drizzle",
	55: "Dense drizzle",
	56: "Light freezing drizzle",
	57: "Dense freezing drizzle",
	61: "Slight rain",
	63: "Moderate rain",
	65: "Heavy rain",
	66: "Light freezing rain",
	67: "Heavy freezing rain",
	71: "Slight snow",
	73: "Moderate snow",
	75: "Heavy snow",
	77: "Snow grains",
	80: "Slight rain showers",
	81: "Moderate rain showers",
	82: "Violent rain showers",
	85: "Slight snow showers",
	86: "Heavy snow showers",
	95: "Thunderstorm",
	96: "Thunderstorm with slight hail",
	99: "Thunderstorm with heavy hail",
}

func WMODescription(code int) string {
	if desc, ok := WMODescriptions[code]; ok {
		return desc
	}
	return fmt.Sprintf("Unknown conditions (WMO %d)", code)
}

type openMeteoGeoResponse struct {
	Results []struct {
		Name      string  `json:"name"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
		Country   string  `json:"country"`
	} `json:"results"`
}

type openMeteoResponse struct {
	UTCOffsetSeconds int `json:"utc_offset_seconds"`
	Current          struct {
		Temperature   float64 `json:"temperature_2m"`
		Apparent      float64 `json:"apparent_temperature"`
		Humidity      float64 `json:"relative_humidity_2m"`
		WeatherCode   int     `json:"weather_code"`
		WindSpeed     float64 `json:"wind_speed_10m"`
		WindDirection float64 `json:"wind_direction_10m"`
	} `json:"current"`
	Hourly struct {
		Time        []int64   `json:"time"`
		Temperature []float64 `json:"temperature_2m"`
		WeatherCode []int     `json:"weather_code"`
	} `json:"hourly"`
	Daily struct {
		Time        []int64   `json:"time"`
		TempMax     []float64 `json:"temperature_2m_max"`
		TempMin     []float64 `json:"temperature_2m_min"`
		WeatherCode []int     `json:"weather_code"`
		Sunrise     []int64   `json:"sunrise"`
		Sunset      []int64   `json:"sunset"`
		UVIndexMax  []float64 `json:"uv_index_max"`
	} `json:"daily"`
}

func fetchOpenMeteo(ctx context.Context, cfg config.Config) (WeatherInfo, error) {
	geoBody, err := GetJSON(ctx, cfg, buildOpenMeteoGeoURL(cfg))
	if err != nil {
		return WeatherInfo{}, err
	}
	lat, lon, err := ParseOpenMeteoGeo(geoBody, cfg.City)
	if err != nil {
		return WeatherInfo{}, err
	}

	body, err := GetJSON(ctx, cfg, BuildOpenMeteoURL(cfg, lat, lon))
	if err != nil {
		return WeatherInfo{}, err
	}
	return parseOpenMeteo(body, cfg)
}

func buildOpenMeteoGeoURL(cfg config.Config) string {
	q := url.Values{}
	q.Set("name", cfg.City)
	q.Set("count", "1")
	return OpenMeteoGeoBase + "?" + q.Encode()
}

func ParseOpenMeteoGeo(body []byte, city string) (lat, lon float64, err error) {
	var r openMeteoGeoResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return 0, 0, fmt.Errorf("failed to decode geocoding response: %w", err)
	}
	if len(r.Results) == 0 {
		return 0, 0, fmt.Errorf("city %q not found", city)
	}
	return r.Results[0].Latitude, r.Results[0].Longitude, nil
}

// OpenMeteoDays returns forecast_days: at least today; two days when hourly
// data must cover the next 24h; capped at the API maximum of 16.
func OpenMeteoDays(cfg config.Config) int {
	days := 1
	if cfg.Forecast > 0 {
		days = min(cfg.Forecast, 16)
	}
	if cfg.Hourly && days < 2 {
		days = 2
	}
	return days
}

func BuildOpenMeteoURL(cfg config.Config, lat, lon float64) string {
	q := url.Values{}
	q.Set("latitude", strconv.FormatFloat(lat, 'f', 4, 64))
	q.Set("longitude", strconv.FormatFloat(lon, 'f', 4, 64))
	q.Set("current", "temperature_2m,apparent_temperature,relative_humidity_2m,weather_code,wind_speed_10m,wind_direction_10m")
	q.Set("daily", "temperature_2m_max,temperature_2m_min,weather_code,sunrise,sunset,uv_index_max")
	if cfg.Hourly {
		q.Set("hourly", "temperature_2m,weather_code")
	}
	q.Set("forecast_days", strconv.Itoa(OpenMeteoDays(cfg)))
	// unixtime keeps time handling zone-safe; utc_offset_seconds recovers the
	// city-local wall clock for display.
	q.Set("timeformat", "unixtime")
	q.Set("timezone", "auto")
	return OpenMeteoBase + "?" + q.Encode()
}

func parseOpenMeteo(body []byte, cfg config.Config) (WeatherInfo, error) {
	var r openMeteoResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	loc := time.FixedZone("city", r.UTCOffsetSeconds)

	desc := WMODescription(r.Current.WeatherCode)
	info := WeatherInfo{
		Description: desc,
		Type:        ClassifyWeather(desc),
		TempC:       r.Current.Temperature,
		TempF:       CToF(r.Current.Temperature),
		FeelsLikeC:  r.Current.Apparent,
		FeelsLikeF:  CToF(r.Current.Apparent),
		Humidity:    r.Current.Humidity,
		WindKph:     r.Current.WindSpeed,
		WindMph:     KphToMph(r.Current.WindSpeed),
		WindDegree:  r.Current.WindDirection,
	}

	if len(r.Daily.UVIndexMax) > 0 {
		info.UVIndex = r.Daily.UVIndexMax[0]
	}
	if len(r.Daily.Sunrise) > 0 && len(r.Daily.Sunset) > 0 {
		info.Sunrise = time.Unix(r.Daily.Sunrise[0], 0).In(loc).Format("15:04")
		info.Sunset = time.Unix(r.Daily.Sunset[0], 0).In(loc).Format("15:04")
	}

	if cfg.Hourly {
		now := time.Now()
		for i, ts := range r.Hourly.Time {
			if i >= len(r.Hourly.Temperature) || i >= len(r.Hourly.WeatherCode) {
				break
			}
			t := time.Unix(ts, 0)
			if t.Before(now.Add(-90*time.Minute)) || t.After(now.Add(24*time.Hour)) {
				continue
			}
			hDesc := WMODescription(r.Hourly.WeatherCode[i])
			info.Hourly = append(info.Hourly, HourlyPoint{
				Time:        t.In(loc),
				TempC:       r.Hourly.Temperature[i],
				TempF:       CToF(r.Hourly.Temperature[i]),
				Description: hDesc,
				Type:        ClassifyWeather(hDesc),
			})
		}
	}

	if cfg.Forecast > 0 {
		for i, ts := range r.Daily.Time {
			if i >= len(r.Daily.TempMax) || i >= len(r.Daily.TempMin) || i >= len(r.Daily.WeatherCode) {
				break
			}
			dDesc := WMODescription(r.Daily.WeatherCode[i])
			info.Forecast = append(info.Forecast, ForecastDay{
				Date:        time.Unix(ts, 0).In(loc),
				MaxTempC:    r.Daily.TempMax[i],
				MaxTempF:    CToF(r.Daily.TempMax[i]),
				MinTempC:    r.Daily.TempMin[i],
				MinTempF:    CToF(r.Daily.TempMin[i]),
				Description: dDesc,
				Type:        ClassifyWeather(dDesc),
			})
		}
	}
	return info, nil
}
