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

	info, err := FetchWeather(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Printf("Weather: %s, %s, UVIndex %s\n", info.Temperature, info.Description, info.UVIndex)

}
