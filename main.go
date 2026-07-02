// Command wrep is a small command-line weather reporter.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/render"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

var version = "dev"

func main() {
	weather.UserAgent = "wrep/" + resolveVersion()

	cfg, err := config.GetConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "wrep:", err)
		os.Exit(1)
	}

	if cfg.ShowVersion {
		fmt.Println("wrep", resolveVersion())
		return
	}

	if cap, ok := weather.ForecastCaps[cfg.APIProvider]; ok && cfg.Forecast > cap && !cfg.Quiet {
		fmt.Fprintf(os.Stderr, "wrep: %s returns at most %d days; truncating\n", cfg.APIProvider, cap)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.Live {
		if err := runLive(ctx, cfg, os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, "wrep:", err)
			os.Exit(1)
		}
		return
	}

	if err := runOnce(ctx, cfg, os.Stdout); err != nil {
		if errors.Is(err, context.Canceled) {
			os.Exit(130)
		}
		fmt.Fprintln(os.Stderr, "wrep:", err)
		os.Exit(1)
	}
}

func runOnce(ctx context.Context, cfg config.Config, out io.Writer) error {
	info, err := weather.FetchWeather(ctx, cfg)
	if err != nil {
		return err
	}
	render.Display(out, info, cfg)
	return nil
}

func runLive(ctx context.Context, cfg config.Config, out io.Writer) error {
	clearScreen := render.StdoutIsTTY() && !cfg.JSON && !cfg.Quiet

	tick := func() {
		if clearScreen {
			fmt.Fprint(out, "\033[H\033[2J")
		} else if !cfg.JSON {
			fmt.Fprintf(out, "--- %s ---\n", time.Now().Format(time.RFC3339))
		}
		if err := runOnce(ctx, cfg, out); err != nil && !errors.Is(err, context.Canceled) {
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
