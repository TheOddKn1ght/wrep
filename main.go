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

	info, err = FetchWeather(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	Display(info, config)
	
}
