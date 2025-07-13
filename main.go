package main

import (
	"fmt"
	"os"
)

func main() {

	config, err := GetConfig()
	if err != nil {
		fmt.Println("Do something about the error: ", err)
	}

	info, err := FetchWeather(config)
	if err != nil {
		fmt.Println("Do something about the error: ", err)
		os.Exit(1)
	}

	fmt.Printf("Weather: %s, %s\n", info.Temperature, info.Description)

}
