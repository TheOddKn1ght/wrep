package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type WeatherInfo struct {
	Temperature string
	Description string
	UVIndex     string
	Forecast    []map[string]interface{}
}

type WttrInResponse struct {
	CurrentCondition []struct {
		Temp_C      string `json:"temp_C"`
		Temp_F      string `json:"temp_F"`
		UvIndex     string `json:"uvIndex"`
		WeatherDesc []struct {
			Value string `json:"value"`
		} `json:"weatherDesc"`
	} `json:"current_condition"`
	Weather []map[string]interface{} `json:"weather"` // for forecast, generic for debug
}

type WeatherAPIResponse struct {
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
				MaxTempC float64 `json:"maxtemp_c"`
				MaxTempF float64 `json:"maxtemp_f"`
				MinTempC float64 `json:"mintemp_c"`
				MinTempF float64 `json:"mintemp_f"`
				Condition struct {
					Text string `json:"text"`
				} `json:"condition"`
			} `json:"day"`
		} `json:"forecastday"`
	} `json:"forecast"`
}

func FetchWeather(config Config) (WeatherInfo, error) {
	var urlStr string
	var parseResponse func([]byte) (WeatherInfo, error)

	switch config.APIProvider {
	case "weatherapi":
		var baseURL string
		if config.Forecast > 0 {
			baseURL = "https://api.weatherapi.com/v1/forecast.json"
		} else {
			baseURL = "https://api.weatherapi.com/v1/current.json"
		}

		u, err := url.Parse(baseURL)
		if err != nil {
			return WeatherInfo{}, fmt.Errorf("failed to parse base URL: %w", err)
		}
		q := u.Query()
		q.Set("key", config.APIKey)
		q.Set("q", config.City)
		if config.Forecast > 0 {
			q.Set("days", fmt.Sprintf("%d", config.Forecast))
		}
		u.RawQuery = q.Encode()
		urlStr = u.String()
		parseResponse = func(body []byte) (WeatherInfo, error) {
			var apiResp WeatherAPIResponse
			if err := json.Unmarshal(body, &apiResp); err != nil {
				return WeatherInfo{}, fmt.Errorf("failed to decode JSON response: %w", err)
			}

			weather := WeatherInfo{
				Description: strings.TrimSpace(apiResp.Current.Condition.Text),
				UVIndex:     fmt.Sprintf("%.1f", apiResp.Current.UVIndex),
				Forecast:    nil,
			}

			if config.Unit == "imperial" {
				weather.Temperature = fmt.Sprintf("%.1f°F", apiResp.Current.TempF)
			} else {
				weather.Temperature = fmt.Sprintf("%.1f°C", apiResp.Current.TempC)
			}

			if config.Forecast > 0 && len(apiResp.Forecast.ForecastDay) > 0 {
				weather.Forecast = make([]map[string]interface{}, len(apiResp.Forecast.ForecastDay))
				for i, day := range apiResp.Forecast.ForecastDay {
					weather.Forecast[i] = map[string]interface{}{
						"date":      day.Date,
						"maxtempC":  fmt.Sprintf("%.1f", day.Day.MaxTempC),
						"maxtempF":  fmt.Sprintf("%.1f", day.Day.MaxTempF),
						"mintempC":  fmt.Sprintf("%.1f", day.Day.MinTempC),
						"mintempF":  fmt.Sprintf("%.1f", day.Day.MinTempF),
						"weatherDesc": []map[string]interface{}{
							{"value": day.Day.Condition.Text},
						},
					}
				}
			}

			return weather, nil
		}
	default:
		u, err := url.Parse("https://wttr.in/" + config.City)
		if err != nil {
			return WeatherInfo{}, fmt.Errorf("failed to parse wttr.in URL: %w", err)
		}
		q := u.Query()
		q.Set("format", "j1")
		u.RawQuery = q.Encode()
		urlStr = u.String()
		parseResponse = func(body []byte) (WeatherInfo, error) {
			var apiResp WttrInResponse
			if err := json.Unmarshal(body, &apiResp); err != nil {
				return WeatherInfo{}, fmt.Errorf("failed to decode JSON response: %w", err)
			}
			if len(apiResp.CurrentCondition) == 0 {
				return WeatherInfo{}, errors.New("no current condition data in response")
			}
			cc := apiResp.CurrentCondition[0]
			weather := WeatherInfo{
				Description: cc.WeatherDesc[0].Value,
				UVIndex:     cc.UvIndex,
				Forecast:    nil,
			}
			if config.Unit == "imperial" {
				weather.Temperature = cc.Temp_F + "°F"
			} else {
				weather.Temperature = cc.Temp_C + "°C"
			}
			if config.Forecast > 0 && len(apiResp.Weather) > 0 {
				weather.Forecast = apiResp.Weather
			}
			return weather, nil
		}
	}

	if config.Verbose {
		fmt.Println("Requesting:", urlStr)
	}

	resp, err := http.Get(urlStr)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if config.APIProvider == "weatherapi" {
		switch resp.StatusCode {
		case http.StatusUnauthorized:
			return WeatherInfo{}, errors.New("unauthorized: invalid or missing API key")
		case http.StatusBadRequest:
			return WeatherInfo{}, errors.New("bad request: city not provided or invalid")
		case http.StatusForbidden:
			return WeatherInfo{}, errors.New("forbidden: API access denied or quota exceeded")
		case http.StatusOK:
			// continue
		default:
			return WeatherInfo{}, fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
		}
	} else {
		if resp.StatusCode != http.StatusOK {
			return WeatherInfo{}, fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to read response body: %w", err)
	}

	return parseResponse(body)
}
