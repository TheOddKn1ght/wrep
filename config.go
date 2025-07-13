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
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("could not determine user home directory: %w", err)
	}

	configPath := filepath.Join(home, ".wrep")

	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		if err := GenerateDefaultConfig(configPath); err != nil {
			return Config{}, fmt.Errorf("failed to generate default config: %w", err)
		}
	}

	f, err := os.Open(configPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	config := Config{}
	scanner := bufio.NewScanner(f)

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
		return Config{}, fmt.Errorf("error reading config file: %w", err)
	}

	if config.APIKey == "" || config.City == "" {
		return Config{}, errors.New("config missing required fields (apiKey and defaultCity)")
	}

	return config, nil
}

func GenerateDefaultConfig(configPath string) error {
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	defaultContent := "apiKey=your_api_key_here\ndefaultCity=Moscow\nunits=metric"
	_, err = f.WriteString(defaultContent)
	return err
}
