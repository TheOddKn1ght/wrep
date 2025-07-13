package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type WeatherInfo struct {
	Temperature string
	Description string
}

type WeatherAPIResponse struct {
	Current struct {
		TempC     float64 `json:"temp_c"`
		TempF     float64 `json:"temp_f"`
		Condition struct {
			Text string `json:"text"`
		} `json:"condition"`
	} `json:"current"`
}

func FetchWeather(config Config) {
	const baseURL = "https://api.weatherapi.com/v1/current.json"

	u, err := url.Parse(baseURL)
	if err != nil {
		fmt.Println("error parsing base URL:", err)
		return
	}
	q := u.Query()
	q.Set("key", config.APIKey)
	q.Set("q", config.City)
	u.RawQuery = q.Encode()

	url := u.String()

	fmt.Println("Requesting:", url)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read error:", err)
		return
	}

	var apiResp WeatherAPIResponse
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		fmt.Println("unmarshal error:", err)
		return
	}

	weather := WeatherInfo{}

	if config.Unit != "imperial" {
		weather = WeatherInfo{
			Temperature: fmt.Sprintf("%.1f°C", apiResp.Current.TempC),
			Description: apiResp.Current.Condition.Text,
		}
	} else {
		weather = WeatherInfo{
			Temperature: fmt.Sprintf("%.1f°F", apiResp.Current.TempF),
			Description: apiResp.Current.Condition.Text,
		}
	}


	fmt.Printf("Weather: %s, %s\n", weather.Temperature, weather.Description)
}
