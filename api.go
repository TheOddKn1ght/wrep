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

func FetchWttrInAPI(config Config) (WeatherInfo, error) {
	baseURL := "https://wttr.in/"
	city := config.City
	u, err := url.Parse(baseURL + city)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to parse wttr.in URL: %w", err)
	}
	q := u.Query()
	q.Set("format", "j1")
	u.RawQuery = q.Encode()
	url := u.String()

	fmt.Println("Requesting:", url)

	resp, err := http.Get(url)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return WeatherInfo{}, fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to read response body: %w", err)
	}

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

	return weather, nil
}

func FetchWeatherAPI(config Config) (WeatherInfo, error) {
	const baseURL = "https://api.weatherapi.com/v1/current.json"

	u, err := url.Parse(baseURL)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to parse base URL: %w", err)
	}

	q := u.Query()
	q.Set("key", config.APIKey)
	q.Set("q", config.City)
	u.RawQuery = q.Encode()
	url := u.String()

	fmt.Println("Requesting: ", url)

	resp, err := http.Get(url)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusUnauthorized:
		return WeatherInfo{}, errors.New("unauthorized: invalid or missing API key")
	case http.StatusBadRequest:
		return WeatherInfo{}, errors.New("bad request: city not provided or invalid")
	case http.StatusForbidden:
		return WeatherInfo{}, errors.New("forbidden: API access denied or quota exceeded")
	case http.StatusOK:

	default:
		return WeatherInfo{}, fmt.Errorf("unexpected HTTP status: %d %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherInfo{}, fmt.Errorf("failed to read response body: %w", err)
	}

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
