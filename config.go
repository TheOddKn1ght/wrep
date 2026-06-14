package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultLiveInterval = 60 * time.Second
	minLiveInterval     = 5 * time.Second
)

type Config struct {
	APIProvider string
	APIKey      string
	City        string
	Unit        string
	Verbose     bool
	Fancy       bool
	NoColor     bool
	JSON        bool
	Quiet       bool
	ShowVersion bool
	Forecast    int
	Live        bool
	Interval    time.Duration
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
		final.Fancy = true
	}
	if cliCfg.Verbose {
		final.Verbose = true
	}
	if cliCfg.NoColor {
		final.NoColor = true
	}
	if cliCfg.JSON {
		final.JSON = true
	}
	if cliCfg.Quiet {
		final.Quiet = true
	}
	if cliCfg.Forecast != 0 {
		final.Forecast = cliCfg.Forecast
	}
	if cliCfg.Live {
		final.Live = true
	}
	if cliCfg.Interval != 0 {
		final.Interval = cliCfg.Interval
	}

	return final
}

func GetConfig() (Config, error) {
	fs := flag.CommandLine
	fs.Usage = usage

	cliConfigDir := flag.String("config", "", "directory containing the .wrep config file (default: $HOME)")
	cliCity := flag.String("city", "", "override city")
	cliUnit := flag.String("unit", "", "override unit: metric or imperial")
	cliAPIKey := flag.String("apikey", "", "override API key (WeatherAPI only)")
	cliAPIProvider := flag.String("apiprovider", "", "API provider: wttr.in or weatherapi")
	cliVerbose := flag.Bool("v", false, "verbose output")
	cliFancy := flag.Bool("fancy", false, "fancy output with colors and emojis")
	cliNoColor := flag.Bool("no-color", false, "disable color escapes (also honors NO_COLOR env)")
	cliJSON := flag.Bool("json", false, "emit raw JSON instead of formatted output")
	cliQuiet := flag.Bool("q", false, "suppress non-error messages")
	cliForecast := flag.Int("f", 0, "show an N-day forecast (e.g. -f 3)")
	cliLive := flag.Bool("live", false, "live mode: refresh weather on an interval until interrupted")
	cliIntervalStr := flag.String("interval", "", "live-mode refresh interval as a Go duration (e.g. 30s, 5m); min 5s")
	cliShowVersion := flag.Bool("V", false, "print version and exit")
	cliShowVersionLong := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *cliShowVersion || *cliShowVersionLong {
		return Config{ShowVersion: true}, nil
	}

	var cliInterval time.Duration
	if s := strings.TrimSpace(*cliIntervalStr); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return Config{}, fmt.Errorf("invalid -interval %q: %w", s, err)
		}
		cliInterval = d
	}

	cliConfig := Config{
		APIProvider: *cliAPIProvider,
		APIKey:      *cliAPIKey,
		City:        *cliCity,
		Unit:        *cliUnit,
		Verbose:     *cliVerbose,
		Fancy:       *cliFancy,
		NoColor:     *cliNoColor,
		JSON:        *cliJSON,
		Quiet:       *cliQuiet,
		Forecast:    *cliForecast,
		Live:        *cliLive,
		Interval:    cliInterval,
	}

	configDir := *cliConfigDir
	if configDir == "" {
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

	fileConfig, err := readConfigFile(configPath)
	if err != nil {
		return Config{}, err
	}

	final := MergeConfig(fileConfig, cliConfig)

	if final.APIProvider == "" {
		final.APIProvider = ProviderWttr
	}
	if final.Unit == "" {
		final.Unit = UnitMetric
	}
	if !validProvider(final.APIProvider) {
		return Config{}, fmt.Errorf("invalid apiProvider %q (want %q or %q)", final.APIProvider, ProviderWttr, ProviderWeatherAPI)
	}
	if !validUnit(final.Unit) {
		return Config{}, fmt.Errorf("invalid unit %q (want %q or %q)", final.Unit, UnitMetric, UnitImperial)
	}
	if final.City == "" {
		return Config{}, errors.New("config missing required field: defaultCity (or pass -city)")
	}
	if final.APIProvider == ProviderWeatherAPI && (final.APIKey == "" || final.APIKey == "your_api_key_here") {
		return Config{}, errors.New("apiProvider=weatherapi requires apiKey (set in ~/.wrep or pass -apikey)")
	}
	if final.JSON && final.Fancy {
		final.Fancy = false
	}
	if final.Live && final.Interval == 0 {
		final.Interval = defaultLiveInterval
	}
	if final.Interval > 0 && final.Interval < minLiveInterval {
		return Config{}, fmt.Errorf("interval must be at least %s (got %s)", minLiveInterval, final.Interval)
	}

	return final, nil
}

func readConfigFile(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
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
			cfg.APIKey = value
		case "defaultCity":
			cfg.City = value
		case "units":
			cfg.Unit = value
		case "apiProvider":
			cfg.APIProvider = value
		case "fancy":
			cfg.Fancy = parseBool(value)
		case "verbose":
			cfg.Verbose = parseBool(value)
		case "noColor":
			cfg.NoColor = parseBool(value)
		case "json":
			cfg.JSON = parseBool(value)
		case "quiet":
			cfg.Quiet = parseBool(value)
		case "live":
			cfg.Live = parseBool(value)
		case "interval":
			d, err := time.ParseDuration(value)
			if err != nil {
				return Config{}, fmt.Errorf("invalid interval %q in config: %w", value, err)
			}
			cfg.Interval = d
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("error reading config file: %w", err)
	}
	return cfg, nil
}

func GenerateDefaultConfig(configPath string) error {
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	const defaultContent = `# wrep config - flags on the command line override these values.
apiKey=your_api_key_here
defaultCity=Moscow
units=metric
apiProvider=wttr.in
fancy=off
verbose=off
noColor=off
live=off
# interval=60s
`
	_, err = f.WriteString(defaultContent)
	return err
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "true", "yes", "1":
		return true
	}
	return false
}

func validProvider(p string) bool {
	return p == ProviderWttr || p == ProviderWeatherAPI
}

func validUnit(u string) bool {
	return u == UnitMetric || u == UnitImperial
}

func usage() {
	out := flag.CommandLine.Output()
	fmt.Fprintln(out, "wrep — a tiny command-line weather reporter.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Usage:")
	fmt.Fprintln(out, "  wrep [flags]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Flags:")
	flag.PrintDefaults()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Examples:")
	fmt.Fprintln(out, "  wrep -city=Berlin -fancy")
	fmt.Fprintln(out, "  wrep -f 3 -fancy")
	fmt.Fprintln(out, "  wrep -apiprovider=weatherapi -apikey=$KEY -city=Tokyo -unit=imperial")
	fmt.Fprintln(out, "  wrep -json | jq")
	fmt.Fprintln(out, "  wrep -live -interval=30s -fancy")
	fmt.Fprintln(out, "  wrep -live -interval=1m -json | jq .")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Environment:")
	fmt.Fprintln(out, "  NO_COLOR   when set (any value), disables color escapes even with -fancy")
}
