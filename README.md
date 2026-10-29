# wrep

Command-line weather in Go: current conditions, hourly and daily forecasts, JSON, and status-bar templates.

```sh
go install github.com/TheOddKn1ght/wrep@latest
wrep -city=Berlin -fancy
wrep -apiprovider=open-meteo -city=Tokyo -f 7
wrep -hourly
wrep -format "{emoji} {temp} {winddir}"
wrep -live -interval=30s
wrep -h
```

Providers: `wttr.in` (default), `open-meteo` (no key), and `weatherapi` (requires `-apikey` or `apiKey` in config).

Defaults live in `~/.wrep`; flags override them. Use `-unit=imperial` for imperial units and `-json` for scripts. Responses are cached for 10 minutes, with stale fallback on failure; `-no-cache` bypasses cache reads.
