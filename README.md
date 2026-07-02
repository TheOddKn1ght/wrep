# wrep

A small command-line weather reporter written in Go. Fetches current weather, hourly and multi-day forecasts from [wttr.in](https://wttr.in), [Open-Meteo](https://open-meteo.com/), or [WeatherAPI](https://www.weatherapi.com/).

## Features
- Three providers: `wttr.in` (default), `open-meteo` (no key, 16-day forecasts), `weatherapi`
- Current weather: temperature, feels-like, humidity, wind speed + compass direction, UV index, sunrise/sunset
- `-hourly`: next 24 hours as a temperature sparkline (`▁▂▄▆█▇▅`) plus a table
- Multi-day forecast as a Unicode table
- `-art` neofetch-style display: weather info next to ASCII art
- `-format "{emoji} {temp}"` custom templates for tmux/i3/polybar status bars
- Response cache with stale fallback — instant repeat runs, and results survive provider outages
- Automatic retry on transient network failures / 5xx
- Plain output by default; `-fancy` adds colors + emoji
- Honors [NO_COLOR](https://no-color.org/) and detects when stdout isn't a TTY
- `-json` mode for piping into `jq` or scripts
- Live mode: refresh on a configurable interval (`-live -interval=30s`)
- Metric / imperial units, configurable via file or CLI

## Install

```sh
git clone https://github.com/TheOddKn1ght/wrep.git
cd wrep
make build        # or: go build
```

Or `go install github.com/TheOddKn1ght/wrep@latest`.

## Providers

| Provider | API key | Forecast | Notes |
|----------|---------|----------|-------|
| `wttr.in` (default) | none | up to 3 days | can be slow when the public instance is under load |
| `open-meteo` | none | up to 16 days | built-in geocoding, hourly data |
| `weatherapi` | required | plan-dependent | needs `apiKey` in config or `-apikey` |

## Usage

```sh
./wrep [flags]
```

### Flags
| Flag | Description |
|------|-------------|
| `-city`         | Override city (e.g. `-city=London`) |
| `-unit`         | `metric` or `imperial` |
| `-apikey`       | WeatherAPI key (overrides config) |
| `-apiprovider`  | `wttr.in`, `open-meteo`, or `weatherapi` |
| `-f`            | Show an N-day forecast (e.g. `-f 3`) |
| `-hourly`       | Show the next 24 hours as a sparkline + table |
| `-fancy`        | Color + emoji output |
| `-art`          | Neofetch-style display: weather info next to ASCII art |
| `-format`       | Custom output template (see [Format templates](#format-templates)) |
| `-no-color`     | Disable color escapes (honors `NO_COLOR` env too) |
| `-json`         | Emit raw JSON instead of formatted output |
| `-no-cache`     | Bypass the response cache |
| `-cache-ttl`    | Cache freshness window as a Go duration (default `10m`) |
| `-v`            | Verbose (prints request URLs and cache hits to stderr) |
| `-q`            | Quiet (suppresses warnings) |
| `-V`, `-version`| Print version and exit |
| `-config`       | Directory containing `.wrep` (default: `$HOME`) |
| `-live`         | Refresh on an interval until interrupted (Ctrl+C to exit) |
| `-interval`     | Refresh interval as a Go duration (e.g. `30s`, `5m`); default `60s`, min `5s` |

### Examples
```sh
./wrep -city=Berlin -fancy
./wrep -f 3 -fancy
./wrep -hourly
./wrep -apiprovider=open-meteo -city=Tokyo -f 7
./wrep -apiprovider=weatherapi -apikey=$KEY -city=Tokyo -unit=imperial
./wrep -art -fancy
./wrep -json | jq '.temp_c'
```

### Hourly mode

`-hourly` shows the next 24 hours: a sparkline of the temperature curve, the min–max range, and a sampled table of times and conditions.

```
Weather: 23.0°C (feels like 25.0°C), Partly cloudy, humidity 57%, wind 18 km/h SW, UVIndex 7.0

Next 24 hours
  ▃▄▆▇█▇▅▄▂▁▁▂  17.0°C – 26.0°C

  15:00   24.0°C  Sunny
  18:00   26.0°C  Sunny
  21:00   21.0°C  Clear
  ...
```

Combine with `-f` to append the daily table below it.

### Art mode

```
    \   /      City: London
     .-.       Condition: Sunny
  - (   ) -    Temperature: 25.0°C
     '-'       Feels like: 25.0°C
    /   \      Humidity: 36%
               Wind: 20 km/h NW
               UV Index: 8.0
               Sunrise: 06:43 AM
               Sunset: 09:12 PM
```

### Format templates

`-format` prints a single plain-text line from a template — built for status bars:

```sh
./wrep -format "{emoji} {temp} {winddir}"     # → ☀️ 25.0°C NW
```

tmux example:

```
set -g status-right '#(wrep -format "{emoji} {temp}")'
```

| Placeholder | Meaning |
|-------------|---------|
| `{city}` | City name |
| `{desc}` | Condition text |
| `{emoji}` | Condition emoji |
| `{temp}` / `{feels}` | Temperature / feels-like in the configured unit |
| `{tempc}` / `{tempf}` | Raw Celsius / Fahrenheit values |
| `{humidity}` | Relative humidity (no `%` sign) |
| `{wind}` | Wind speed in the configured unit |
| `{winddir}` | Wind compass direction (`N`, `SSW`, ...) |
| `{uv}` | UV index |
| `{sunrise}` / `{sunset}` | Sun times (when the provider supplies them) |
| `{min}` / `{max}` | Today's min/max (requires `-f`) |

Unknown placeholders are left as-is.

### Caching

Responses are cached under the user cache directory (`~/Library/Caches/wrep` on macOS, `~/.cache/wrep` on Linux) for 10 minutes by default:

- Repeat runs within the TTL are instant and don't hit the provider.
- If a fetch fails, wrep falls back to the most recent cached response (however old) and prints a warning — handy when wttr.in is having a moment.
- `-live` mode always refetches (that's the point), but still benefits from the stale fallback.
- `-no-cache` bypasses reads; `-cache-ttl 1h` widens the window.

### Live mode

`-live` re-fetches and re-renders on the interval set by `-interval` (default `60s`, minimum `5s`). Ctrl+C exits cleanly; transient fetch failures print a stderr warning and the loop keeps going.

```sh
./wrep -live -interval=30s -fancy        # dashboard: clears screen each tick
./wrep -live -interval=1m -json | jq .   # NDJSON: one JSON object per tick
```

When stdout is a TTY and neither `-json` nor `-q` is set, the screen is cleared and redrawn each tick. Otherwise (piped output, JSON, or quiet) each tick is appended below the previous one — for `-json` this means newline-delimited JSON suitable for piping.

### Environment
- `NO_COLOR` — when set to any non-empty value, color escapes are suppressed even with `-fancy`.

## Configuration

On first run, a config file is created at `~/.wrep` (mode `0600`, since it may hold your API key):

```
apiKey=your_api_key_here
defaultCity=Moscow
units=metric
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
```

Edit it to set your defaults. CLI flags override file values.

| Key | Values |
|-----|--------|
| `apiKey`      | Your WeatherAPI key (not required for wttr.in / open-meteo) |
| `defaultCity` | Default city |
| `units`       | `metric` or `imperial` |
| `apiProvider` | `wttr.in`, `open-meteo`, or `weatherapi` |
| `fancy`       | `on` / `off` (also accepts `true`/`false`/`yes`/`1`) |
| `verbose`     | `on` / `off` |
| `noColor`     | `on` / `off` |
| `live`        | `on` / `off` — enable live refresh mode |
| `art`         | `on` / `off` — enable art mode |
| `hourly`      | `on` / `off` — enable hourly mode |
| `cache`       | `on` / `off` — response caching (default on) |
| `cacheTTL`    | Cache freshness window (e.g. `10m`, `1h`) |
| `format`      | Output template (see [Format templates](#format-templates)) |
| `interval`    | Go duration string (e.g. `30s`, `5m`); min `5s` |

## Development

```sh
make build     # build with version stamped from git
make test      # go test -race ./...
make lint      # gofmt + go vet
```

CI runs lint, build, and the race-enabled test suite on Linux, macOS, and Windows.
