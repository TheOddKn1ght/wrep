package main

import (
	"fmt"
)

func main() {

	config, err := GetConfig()

	if err != nil {
		fmt.Println("Do something about the error")
	}

	FetchWeather(config)

}
