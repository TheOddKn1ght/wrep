package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {

	fmt.Println("Hello")
	city := flag.String("city", "Moscow", "specify the city for the weather forecast")
	unit := flag.String("unit", "C", "specify the unit for the weather forecast C°/F°")
	flag.Parse()
	fmt.Println(*city)
	fmt.Println(*unit)

	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".wrep")

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("config exists")

		f, err := os.Open(configPath)

		if err != nil {
			fmt.Println("error opening file")
		}

		defer f.Close()

		scanner := bufio.NewScanner(f)

		config := Config{}

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())

			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			switch key {
			case "apiKey":
				config.APIKey = value
			case "defaultCity":
				config.City = value
			case "units":
				config.Unit = value
			}

		}

		if err := scanner.Err(); err != nil {
			fmt.Println("error reading file:", err)
		}

		fmt.Println(config)

		FetchWeather(config)

	} else if errors.Is(err, os.ErrNotExist) {
		fmt.Println("does not exist")
		f, err := os.Create(configPath)
		if err != nil {
			fmt.Println("could not create the config file")
			os.Exit(1)
		}

		defer f.Close()

		f.WriteString("apiKey=your_api_key_here\ndefaultCity=Moscow\nunits=c\n")

		if err != nil {
			fmt.Println("could not write to the config file")
			os.Exit(1)
		}
	}
}
