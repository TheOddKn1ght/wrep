package main

import (
	"flag"
	"fmt"
)

func main() {

	city := flag.String("city", "Moscow", "specify the city for the weather forecast")
	unit := flag.String("unit", "C", "specify the unit for the weather forecast C°/F°")
	flag.Parse()
	fmt.Println(*city)
	fmt.Println(*unit)

	config, err := GetConfig()

	if err != nil {
		fmt.Println("Do something about the error")
	}

	FetchWeather(config)

}
