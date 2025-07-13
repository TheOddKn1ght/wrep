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

type Config struct {
	APIKey string
	City   string
	Unit   string
}

func MergeConfig(fileCfg Config, cliCfg Config) Config {
	final := fileCfg

	if cliCfg.APIKey != "" {
		final.APIKey = cliCfg.APIKey
	}

	if cliCfg.City != "" {
		final.City = cliCfg.City
	}

	if cliCfg.Unit != "" {
		final.Unit = cliCfg.Unit
	}

	return final
}

func GetConfig() (Config, error) {

	cliCity := flag.String("city", "", "override city from CLI")
	cliUnit := flag.String("unit", "", "override unit from CLI")
	cliAPIKey := flag.String("apikey", "", "override API key from CLI")
	flag.Parse()

	cliConfig := Config{
		APIKey: *cliAPIKey,
		City:   *cliCity,
		Unit:   *cliUnit,
	}

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

	fileConfig := Config{}
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
			fileConfig.APIKey = value
		case "defaultCity":
			fileConfig.City = value
		case "units":
			fileConfig.Unit = value
		}
	}

	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("error reading config file: %w", err)
	}

	if fileConfig.APIKey == "" || fileConfig.City == "" {
		return Config{}, errors.New("config missing required fields (apiKey and defaultCity)")
	}

	final := MergeConfig(fileConfig, cliConfig)

	return final, nil
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
