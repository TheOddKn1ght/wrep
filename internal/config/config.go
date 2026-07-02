// Package config owns the wrep configuration: CLI flags, the ~/.wrep file,
// merging, validation, and the shared provider/unit constants.
package config

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
	ProviderWttr       = "wttr.in"
	ProviderWeatherAPI = "weatherapi"
	ProviderOpenMeteo  = "open-meteo"
	UnitMetric         = "metric"
	UnitImperial       = "imperial"
)

const (
	defaultLiveInterval = 60 * time.Second
	minLiveInterval     = 5 * time.Second
	DefaultCacheTTL     = 10 * time.Minute
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
	Hourly      bool
	Live        bool
	Art         bool
	Interval    time.Duration
	NoCache     bool
	CacheTTL    time.Duration
	Format      string
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
	if cliCfg.Art {
		final.Art = true
	}
	if cliCfg.Hourly {
		final.Hourly = true
	}
	if cliCfg.Interval != 0 {
		final.Interval = cliCfg.Interval
	}
	if cliCfg.NoCache {
		final.NoCache = true
	}
	if cliCfg.CacheTTL != 0 {
		final.CacheTTL = cliCfg.CacheTTL
	}
	if cliCfg.Format != "" {
		final.Format = cliCfg.Format
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
	cliAPIProvider := flag.String("apiprovider", "", "API provider: wttr.in, open-meteo or weatherapi")
	cliVerbose := flag.Bool("v", false, "verbose output")
	cliFancy := flag.Bool("fancy", false, "fancy output with colors and emojis")
	cliNoColor := flag.Bool("no-color", false, "disable color escapes (also honors NO_COLOR env)")
	cliJSON := flag.Bool("json", false, "emit raw JSON instead of formatted output")
	cliQuiet := flag.Bool("q", false, "suppress non-error messages")
	cliForecast := flag.Int("f", 0, "show an N-day forecast (e.g. -f 3)")
	cliHourly := flag.Bool("hourly", false, "show the next 24 hours as a sparkline + table")
	cliLive := flag.Bool("live", false, "live mode: refresh weather on an interval until interrupted")
	cliArt := flag.Bool("art", false, "neofetch-style display: weather info next to ASCII art")
	cliIntervalStr := flag.String("interval", "", "live-mode refresh interval as a Go duration (e.g. 30s, 5m); min 5s")
	cliNoCache := flag.Bool("no-cache", false, "bypass the response cache")
	cliCacheTTLStr := flag.String("cache-ttl", "", "cache freshness window as a Go duration (default 10m)")
	cliFormat := flag.String("format", "", "custom output template, e.g. \"{emoji} {temp}\" (see README for placeholders)")
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

	var cliCacheTTL time.Duration
	if s := strings.TrimSpace(*cliCacheTTLStr); s != "" {
		d, err := time.ParseDuration(s)
		if err != nil {
			return Config{}, fmt.Errorf("invalid -cache-ttl %q: %w", s, err)
		}
		cliCacheTTL = d
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
		Hourly:      *cliHourly,
		Live:        *cliLive,
		Art:         *cliArt,
		Interval:    cliInterval,
		NoCache:     *cliNoCache,
		CacheTTL:    cliCacheTTL,
		Format:      *cliFormat,
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

	fileConfig, err := ReadConfigFile(configPath)
	if err != nil {
		return Config{}, err
	}

	final := MergeConfig(fileConfig, cliConfig)

	if final.APIProvider == "" {
		final.APIProvider = ProviderWttr
	}
	if final.APIProvider == "openmeteo" {
		final.APIProvider = ProviderOpenMeteo
	}
	if final.Unit == "" {
		final.Unit = UnitMetric
	}
	if !ValidProvider(final.APIProvider) {
		return Config{}, fmt.Errorf("invalid apiProvider %q (want %q, %q or %q)", final.APIProvider, ProviderWttr, ProviderWeatherAPI, ProviderOpenMeteo)
	}
	if !ValidUnit(final.Unit) {
		return Config{}, fmt.Errorf("invalid unit %q (want %q or %q)", final.Unit, UnitMetric, UnitImperial)
	}
	if final.City == "" {
		return Config{}, errors.New("config missing required field: defaultCity (or pass -city)")
	}
	if final.Forecast < 0 {
		return Config{}, fmt.Errorf("forecast days must be positive (got %d)", final.Forecast)
	}
	if final.APIProvider == ProviderWeatherAPI && (final.APIKey == "" || final.APIKey == "your_api_key_here") {
		return Config{}, errors.New("apiProvider=weatherapi requires apiKey (set in ~/.wrep or pass -apikey)")
	}
	if final.JSON && final.Fancy {
		final.Fancy = false
	}
	if final.JSON && final.Art {
		final.Art = false
	}
	if final.JSON && final.Format != "" {
		final.Format = ""
	}
	if final.Art && final.Forecast > 0 {
		if !final.Quiet {
			fmt.Fprintln(os.Stderr, "wrep: -art does not support -f; ignoring forecast")
		}
		final.Forecast = 0
	}
	if final.Art && final.Hourly {
		if !final.Quiet {
			fmt.Fprintln(os.Stderr, "wrep: -art does not support -hourly; ignoring hourly")
		}
		final.Hourly = false
	}
	if final.Live && final.Interval == 0 {
		final.Interval = defaultLiveInterval
	}
	if final.Interval > 0 && final.Interval < minLiveInterval {
		return Config{}, fmt.Errorf("interval must be at least %s (got %s)", minLiveInterval, final.Interval)
	}
	if final.CacheTTL < 0 {
		return Config{}, fmt.Errorf("cache TTL must be positive (got %s)", final.CacheTTL)
	}
	if final.CacheTTL == 0 {
		final.CacheTTL = DefaultCacheTTL
	}

	return final, nil
}

func ReadConfigFile(path string) (Config, error) {
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
			cfg.Fancy = ParseBool(value)
		case "verbose":
			cfg.Verbose = ParseBool(value)
		case "noColor":
			cfg.NoColor = ParseBool(value)
		case "json":
			cfg.JSON = ParseBool(value)
		case "quiet":
			cfg.Quiet = ParseBool(value)
		case "live":
			cfg.Live = ParseBool(value)
		case "art":
			cfg.Art = ParseBool(value)
		case "hourly":
			cfg.Hourly = ParseBool(value)
		case "cache":
			cfg.NoCache = !ParseBool(value)
		case "format":
			cfg.Format = value
		case "interval":
			d, err := time.ParseDuration(value)
			if err != nil {
				return Config{}, fmt.Errorf("invalid interval %q in config: %w", value, err)
			}
			cfg.Interval = d
		case "cacheTTL":
			d, err := time.ParseDuration(value)
			if err != nil {
				return Config{}, fmt.Errorf("invalid cacheTTL %q in config: %w", value, err)
			}
			cfg.CacheTTL = d
		}
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("error reading config file: %w", err)
	}
	return cfg, nil
}

func GenerateDefaultConfig(configPath string) error {
	// 0600: the config may hold a WeatherAPI key.
	f, err := os.OpenFile(configPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()

	const defaultContent = `# wrep config - flags on the command line override these values.
apiKey=your_api_key_here
defaultCity=Moscow
units=metric
# apiProvider: wttr.in, weatherapi, or open-meteo
apiProvider=wttr.in
fancy=off
verbose=off
noColor=off
live=off
art=off
hourly=off
cache=on
# interval=60s
# cacheTTL=10m
# format={emoji} {temp}
`
	_, err = f.WriteString(defaultContent)
	return err
}

func ParseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "on", "true", "yes", "1":
		return true
	}
	return false
}

func ValidProvider(p string) bool {
	return p == ProviderWttr || p == ProviderWeatherAPI || p == ProviderOpenMeteo
}

func ValidUnit(u string) bool {
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
	fmt.Fprintln(out, "  wrep -hourly")
	fmt.Fprintln(out, "  wrep -apiprovider=open-meteo -city=Tokyo -f 7")
	fmt.Fprintln(out, "  wrep -apiprovider=weatherapi -apikey=$KEY -city=Tokyo -unit=imperial")
	fmt.Fprintln(out, "  wrep -json | jq")
	fmt.Fprintln(out, "  wrep -live -interval=30s -fancy")
	fmt.Fprintln(out, "  wrep -art -fancy")
	fmt.Fprintln(out, "  wrep -format \"{emoji} {temp} {winddir}\"   # for tmux/i3/polybar status bars")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Environment:")
	fmt.Fprintln(out, "  NO_COLOR   when set (any value), disables color escapes even with -fancy")
}
