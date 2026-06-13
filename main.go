package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

var version = "dev"

func main() {
	UserAgent = "wrep/" + resolveVersion()

	config, err := GetConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wrep:", err)
		os.Exit(1)
	}

	if config.ShowVersion {
		fmt.Println("wrep", resolveVersion())
		return
	}

	if config.APIProvider == ProviderWttr && config.Forecast > 3 && !config.Quiet {
		fmt.Fprintln(os.Stderr, "wrep: wttr.in returns at most 3 days; truncating")
	}

	info, err := FetchWeather(config)
	if err != nil {
		fmt.Fprintln(os.Stderr, "wrep:", err)
		os.Exit(1)
	}

	Display(os.Stdout, info, config)
}

func resolveVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}
	return version
}
