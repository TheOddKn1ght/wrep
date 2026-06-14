package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"
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

	if config.Live {
		if err := runLive(config, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "wrep:", err)
			os.Exit(1)
		}
		return
	}

	if err := runOnce(config, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "wrep:", err)
		os.Exit(1)
	}
}

func runOnce(cfg Config, out io.Writer) error {
	info, err := FetchWeather(cfg)
	if err != nil {
		return err
	}
	Display(out, info, cfg)
	return nil
}

func runLive(cfg Config, out io.Writer) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	clearScreen := stdoutIsTTY() && !cfg.JSON && !cfg.Quiet

	tick := func() {
		if clearScreen {
			fmt.Fprint(out, "\033[H\033[2J")
		} else if !cfg.JSON {
			fmt.Fprintf(out, "--- %s ---\n", time.Now().Format(time.RFC3339))
		}
		if err := runOnce(cfg, out); err != nil {
			fmt.Fprintln(os.Stderr, "wrep:", err)
		}
	}

	tick()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			tick()
		}
	}
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
