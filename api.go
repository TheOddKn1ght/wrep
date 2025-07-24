package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type WeatherInfo struct {
	Temperature string
	Description string
	UVIndex     string
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
}

func FetchWeather(config Config) (WeatherInfo, error) {
	var urlStr string
	var parseResponse func([]byte) (WeatherInfo, error)

	switch config.APIProvider {
	case "weatherapi":
		u, err := url.Parse("https://api.weatherapi.com/v1/current.json")
		if err != nil {
			return WeatherInfo{}, fmt.Errorf("failed to parse base URL: %w", err)
		}
		q := u.Query()
		q.Set("key", config.APIKey)
		q.Set("q", config.City)
		u.RawQuery = q.Encode()
		urlStr = u.String()
		parseResponse = func(body []byte) (WeatherInfo, error) {
			var apiResp WeatherAPIResponse
			if err := json.Unmarshal(body, &apiResp); err != nil {
				return WeatherInfo{}, fmt.Errorf("failed to decode JSON response: %w", err)
			}
			weather := WeatherInfo{
				Description: apiResp.Current.Condition.Text,
				UVIndex:     fmt.Sprintf("%.1f", apiResp.Current.UVIndex),
			}
			if config.Unit == "imperial" {
				weather.Temperature = fmt.Sprintf("%.1f°F", apiResp.Current.TempF)
			} else {
				weather.Temperature = fmt.Sprintf("%.1f°C", apiResp.Current.TempC)
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
			}
			if config.Unit == "imperial" {
				weather.Temperature = cc.Temp_F + "°F"
			} else {
				weather.Temperature = cc.Temp_C + "°C"
			}
			// debug output
			if config.Forecast > 0 && len(apiResp.Weather) > 0 {
				fmt.Println("Forecast:")
				for i := 0; i < config.Forecast && i < len(apiResp.Weather); i++ {
					w := apiResp.Weather[i]
					date, _ := w["date"].(string)
					maxtempc, _ := w["maxtempC"].(string)
					maxtempf, _ := w["maxtempF"].(string)
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
					fmt.Printf("Day %d: %s, MaxC: %s, MaxF: %s, Desc: %s\n", i+1, date, maxtempc, maxtempf, desc)
				}
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
