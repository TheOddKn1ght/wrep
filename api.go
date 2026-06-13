package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ProviderWttr       = "wttr.in"
	ProviderWeatherAPI = "weatherapi"
	UnitMetric         = "metric"
	UnitImperial       = "imperial"
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

type WeatherInfo struct {
	TempC       float64       `json:"temp_c"`
	TempF       float64       `json:"temp_f"`
	UVIndex     float64       `json:"uv_index"`
	Description string        `json:"description"`
	Type        WeatherType   `json:"-"`
	Forecast    []ForecastDay `json:"forecast,omitempty"`
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

var (
	UserAgent  = "wrep"
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

type wttrInDesc struct {
	Value string `json:"value"`
}

type wttrInResponse struct {
	CurrentCondition []struct {
		TempC       string       `json:"temp_C"`
		TempF       string       `json:"temp_F"`
		UvIndex     string       `json:"uvIndex"`
		WeatherDesc []wttrInDesc `json:"weatherDesc"`
	} `json:"current_condition"`
	Weather []struct {
		Date     string `json:"date"`
		MaxTempC string `json:"maxtempC"`
		MaxTempF string `json:"maxtempF"`
		MinTempC string `json:"mintempC"`
		MinTempF string `json:"mintempF"`
		Hourly   []struct {
			WeatherDesc []wttrInDesc `json:"weatherDesc"`
		} `json:"hourly"`
	} `json:"weather"`
}

type weatherAPIResponse struct {
	Current struct {
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		UVIndex   float64 `json:"uv"`
		Condition struct {
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
		} `json:"forecastday"`
	} `json:"forecast"`
}

func FetchWeather(config Config) (WeatherInfo, error) {
	urlStr, err := buildURL(config)
	if err != nil {
		return WeatherInfo{}, err
	}

	if config.Verbose && !config.Quiet {
		fmt.Fprintln(os.Stderr, "Requesting:", urlStr)
	}

	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to build request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if err := checkStatus(resp, config.APIProvider); err != nil {
		return WeatherInfo{}, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to read response body: %w", err)
	}

	switch config.APIProvider {
	case ProviderWeatherAPI:
		return parseWeatherAPI(body, config)
	default:
		return parseWttr(body, config)
	}
}

func buildURL(config Config) (string, error) {
	switch config.APIProvider {
	case ProviderWeatherAPI:
		base := "https://api.weatherapi.com/v1/current.json"
		if config.Forecast > 0 {
			base = "https://api.weatherapi.com/v1/forecast.json"
		}
		u, err := url.Parse(base)
		if err != nil {
			return "", fmt.Errorf("failed to parse base URL: %w", err)
		}
		q := u.Query()
		q.Set("key", config.APIKey)
		q.Set("q", config.City)
		if config.Forecast > 0 {
			q.Set("days", strconv.Itoa(config.Forecast))
		}
		u.RawQuery = q.Encode()
		return u.String(), nil
	default:
		u, err := url.Parse("https://wttr.in/" + url.PathEscape(config.City))
		if err != nil {
			return "", fmt.Errorf("failed to parse wttr.in URL: %w", err)
		}
		q := u.Query()
		q.Set("format", "j1")
		u.RawQuery = q.Encode()
		return u.String(), nil
	}
}

func checkStatus(resp *http.Response, provider string) error {
	if provider == ProviderWeatherAPI {
		switch resp.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusUnauthorized:
			return errors.New("unauthorized: invalid or missing API key")
		case http.StatusBadRequest:
			return errors.New("bad request: city not provided or invalid")
		case http.StatusForbidden:
			return errors.New("forbidden: API access denied or quota exceeded")
		default:
			return fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
		}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
	}
	return nil
}

func parseWttr(body []byte, config Config) (WeatherInfo, error) {
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
		TempC:       parseFloat(cc.TempC),
		TempF:       parseFloat(cc.TempF),
		UVIndex:     parseFloat(cc.UvIndex),
	}
	info.Type = ClassifyWeather(info.Description)

	if config.Forecast > 0 {
		for _, day := range r.Weather {
			desc := representativeWttrDesc(day.Hourly)
			d, _ := time.Parse("2006-01-02", day.Date)
			info.Forecast = append(info.Forecast, ForecastDay{
				Date:        d,
				MaxTempC:    parseFloat(day.MaxTempC),
				MinTempC:    parseFloat(day.MinTempC),
				MaxTempF:    parseFloat(day.MaxTempF),
				MinTempF:    parseFloat(day.MinTempF),
				Description: desc,
				Type:        ClassifyWeather(desc),
			})
		}
	}
	return info, nil
}

func parseWeatherAPI(body []byte, config Config) (WeatherInfo, error) {
	var r weatherAPIResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to decode JSON response: %w", err)
	}
	info := WeatherInfo{
		Description: strings.TrimSpace(r.Current.Condition.Text),
		TempC:       r.Current.TempC,
		TempF:       r.Current.TempF,
		UVIndex:     r.Current.UVIndex,
	}
	info.Type = ClassifyWeather(info.Description)
	if config.Forecast > 0 {
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

// representativeWttrDesc picks the noon entry (3-hour interval, index 4) from
// wttr.in's hourly slice so the forecast row shows a midday condition rather
// than midnight. Falls back to whatever's available.
func representativeWttrDesc(hourly []struct {
	WeatherDesc []wttrInDesc `json:"weatherDesc"`
}) string {
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

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return f
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
