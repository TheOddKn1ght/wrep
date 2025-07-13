package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIKey string
	City   string
	Unit   string
}

func GetConfig() (Config, error) {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".wrep")

	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("config exists")

		f, err := os.Open(configPath)

		if err != nil {
			fmt.Println("Error opening file")
			return Config{}, nil
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
			fmt.Println("Error reading file:", err)
			return Config{}, err
		}

		fmt.Println(config)
		return config, nil

	} else if errors.Is(err, os.ErrNotExist) {
		GenerateDefaultConfig(configPath)
	}

	return Config{}, nil
}

func GenerateDefaultConfig(configPath string) {
	fmt.Println("does not exist")
	f, err := os.Create(configPath)
	if err != nil {
		fmt.Println("Could not create the config file")
		os.Exit(1)
	}

	defer f.Close()

	f.WriteString("apiKey=your_api_key_here\ndefaultCity=Moscow\nunits=c\n")

	if err != nil {
		fmt.Println("Could not write to the config file")
		os.Exit(1)
	}
}
