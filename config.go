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
	APIProvider string
	APIKey      string
	City        string
	Unit        string
	Verbose     bool
	Fancy       bool
	Forecast    int
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

	if cliCfg.APIProvider != "" {
		final.APIProvider = cliCfg.APIProvider
	}

	if cliCfg.Fancy {
		final.Fancy = cliCfg.Fancy
	}

	if cliCfg.Verbose {
		final.Verbose = cliCfg.Verbose
	}

	if cliCfg.Forecast != 0 {
		final.Forecast = cliCfg.Forecast
	}

	return final
}

func GetConfig() (Config, error) {
	cliConfigDir := flag.String("config", "", "directory path for config file")
	cliCity := flag.String("city", "", "override city from CLI")
	cliUnit := flag.String("unit", "", "override unit from CLI")
	cliAPIKey := flag.String("apikey", "", "override API key from CLI")
	cliAPIProvider := flag.String("apiprovider", "", "API provider to use: wttr.in or weatherapi")
	cliVerbose := flag.Bool("v", false, "verbose")
	cliFancy := flag.Bool("fancy", false, "fancy output with emojis")
	cliForecast := flag.Int("f", 0, "number of days to display forecast for (wttr.in only)")
	flag.Parse()

	cliConfig := Config{
		APIProvider: *cliAPIProvider,
		APIKey:      *cliAPIKey,
		City:        *cliCity,
		Unit:        *cliUnit,
		Verbose:     *cliVerbose,
		Fancy:       *cliFancy,
		Forecast:    *cliForecast,
	}

	var configDir string
	if *cliConfigDir != "" {
		configDir = *cliConfigDir
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return Config{}, fmt.Errorf("could not determine user home directory: %w", err)
		}
		configDir = home
	}

	configPath := filepath.Join(configDir, ".wrep")

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
		case "apiProvider":
			fileConfig.APIProvider = value
		case "fancy":
			if value == "on" {
				fileConfig.Fancy = true
			}
		case "verbose":
			if value == "on" {
				fileConfig.Verbose = true
			}
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

	const defaultContent = "apiKey=your_api_key_here\ndefaultCity=Moscow\nunits=metric\napiProvider=wttr.in\nverbose=off\nfancy=off"
	_, err = f.WriteString(defaultContent)
	return err
}
