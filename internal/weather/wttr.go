package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
)

var WttrBase = "https://wttr.in"

type wttrInDesc struct {
	Value string `json:"value"`
}

type wttrHour struct {
	Time        string       `json:"time"`
	TempC       string       `json:"tempC"`
	TempF       string       `json:"tempF"`
	WeatherDesc []wttrInDesc `json:"weatherDesc"`
}

type wttrDay struct {
	Date      string `json:"date"`
	MaxTempC  string `json:"maxtempC"`
	MaxTempF  string `json:"maxtempF"`
	MinTempC  string `json:"mintempC"`
	MinTempF  string `json:"mintempF"`
	Astronomy []struct {
		Sunrise string `json:"sunrise"`
		Sunset  string `json:"sunset"`
	} `json:"astronomy"`
	Hourly []wttrHour `json:"hourly"`
}

type wttrInResponse struct {
	CurrentCondition []struct {
		TempC            string       `json:"temp_C"`
		TempF            string       `json:"temp_F"`
		FeelsLikeC       string       `json:"FeelsLikeC"`
		FeelsLikeF       string       `json:"FeelsLikeF"`
		Humidity         string       `json:"humidity"`
		WindspeedKmph    string       `json:"windspeedKmph"`
		WindspeedMiles   string       `json:"windspeedMiles"`
		WinddirDegree    string       `json:"winddirDegree"`
		UvIndex          string       `json:"uvIndex"`
		LocalObsDateTime string       `json:"localObsDateTime"`
		WeatherDesc      []wttrInDesc `json:"weatherDesc"`
	} `json:"current_condition"`
	Weather []wttrDay `json:"weather"`
}

func fetchWttr(ctx context.Context, cfg config.Config) (WeatherInfo, error) {
	urlStr, err := BuildWttrURL(cfg)
	if err != nil {
		return WeatherInfo{}, err
	}
	body, err := GetJSON(ctx, cfg, urlStr)
	if err != nil {
		return WeatherInfo{}, err
	}
	return ParseWttr(body, cfg)
}

func BuildWttrURL(cfg config.Config) (string, error) {
	u, err := url.Parse(WttrBase + "/" + url.PathEscape(cfg.City))
	if err != nil {
		return "", fmt.Errorf("failed to parse wttr.in URL: %w", err)
	}
	q := u.Query()
	q.Set("format", "j1")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func ParseWttr(body []byte, cfg config.Config) (WeatherInfo, error) {
	var r wttrInResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	if len(r.CurrentCondition) == 0 {
		return WeatherInfo{}, errors.New("no current condition data in response")
	}
	cc := r.CurrentCondition[0]
	if len(cc.WeatherDesc) == 0 {
		return WeatherInfo{}, errors.New("no weather description in response")
	}

	info := WeatherInfo{
		Description: strings.TrimSpace(cc.WeatherDesc[0].Value),
		TempC:       ParseFloat(cc.TempC),
		TempF:       ParseFloat(cc.TempF),
		FeelsLikeC:  ParseFloat(cc.FeelsLikeC),
		FeelsLikeF:  ParseFloat(cc.FeelsLikeF),
		Humidity:    ParseFloat(cc.Humidity),
		WindKph:     ParseFloat(cc.WindspeedKmph),
		WindMph:     ParseFloat(cc.WindspeedMiles),
		WindDegree:  ParseFloat(cc.WinddirDegree),
		UVIndex:     ParseFloat(cc.UvIndex),
	}
	info.Type = ClassifyWeather(info.Description)

	if len(r.Weather) > 0 && len(r.Weather[0].Astronomy) > 0 {
		info.Sunrise = r.Weather[0].Astronomy[0].Sunrise
		info.Sunset = r.Weather[0].Astronomy[0].Sunset
	}

	if cfg.Hourly {
		info.Hourly = wttrHourlyPoints(r.Weather, WttrCityNow(cc.LocalObsDateTime))
	}

	if cfg.Forecast > 0 {
		for _, day := range r.Weather {
			desc := representativeWttrDesc(day.Hourly)
			d, _ := time.Parse("2006-01-02", day.Date)
			info.Forecast = append(info.Forecast, ForecastDay{
				Date:        d,
				MaxTempC:    ParseFloat(day.MaxTempC),
				MinTempC:    ParseFloat(day.MinTempC),
				MaxTempF:    ParseFloat(day.MaxTempF),
				MinTempF:    ParseFloat(day.MinTempF),
				Description: desc,
				Type:        ClassifyWeather(desc),
			})
		}
	}
	return info, nil
}

// WttrCityNow parses the observation timestamp so hourly filtering happens in
// the city's local time, not the user's. Falls back to the local clock.
func WttrCityNow(localObs string) time.Time {
	if t, err := time.Parse("2006-01-02 03:04 PM", strings.TrimSpace(localObs)); err == nil {
		return t
	}
	return time.Now()
}

// wttrHourlyPoints flattens wttr.in's per-day 3-hourly entries (time encoded
// as "0", "300", ..., "2100" meaning hhmm) into points within the next 24h.
func wttrHourlyPoints(days []wttrDay, cityNow time.Time) []HourlyPoint {
	var pts []HourlyPoint
	for _, day := range days {
		d, err := time.Parse("2006-01-02", day.Date)
		if err != nil {
			continue
		}
		for _, h := range day.Hourly {
			hhmm, err := strconv.Atoi(strings.TrimSpace(h.Time))
			if err != nil {
				continue
			}
			t := d.Add(time.Duration(hhmm/100)*time.Hour + time.Duration(hhmm%100)*time.Minute)
			if t.Before(cityNow.Add(-90*time.Minute)) || t.After(cityNow.Add(24*time.Hour)) {
				continue
			}
			desc := ""
			if len(h.WeatherDesc) > 0 {
				desc = strings.TrimSpace(h.WeatherDesc[0].Value)
			}
			pts = append(pts, HourlyPoint{
				Time:        t,
				TempC:       ParseFloat(h.TempC),
				TempF:       ParseFloat(h.TempF),
				Description: desc,
				Type:        ClassifyWeather(desc),
			})
		}
	}
	return pts
}

// representativeWttrDesc picks the noon entry (3-hour interval, index 4) from
// wttr.in's hourly slice so the forecast row shows a midday condition rather
// than midnight. Falls back to whatever's available.
func representativeWttrDesc(hourly []wttrHour) string {
	if len(hourly) == 0 {
		return ""
	}
	idx := 4
	if idx >= len(hourly) {
		idx = 0
	}
	if len(hourly[idx].WeatherDesc) == 0 {
		return ""
	}
	return strings.TrimSpace(hourly[idx].WeatherDesc[0].Value)
}
