package main

import (
	"fmt"
	"os"
)

func main() {

	config, err := GetConfig()
	if err != nil {
		fmt.Println(err)
	}

	var info WeatherInfo

	switch config.APIProvider {
	case "weatherapi":
		info, err = FetchWeatherAPI(config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	default:
		info, err = FetchWttrInAPI(config)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	Display(info)
}
