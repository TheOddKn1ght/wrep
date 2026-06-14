# wrep

A small command-line weather reporter written in Go. Fetches current weather and an optional multi-day forecast from either [wttr.in](https://wttr.in) (no key required) or [WeatherAPI](https://www.weatherapi.com/).

## Features
- Two providers: `wttr.in` (default) and `weatherapi`
- Current weather: temperature, description, UV index
- Multi-day forecast as a Unicode table
- Plain output by default; `-fancy` adds colors + emoji
- Honors [NO_COLOR](https://no-color.org/) and detects when stdout isn't a TTY
- `-json` mode for piping into `jq` or scripts
- Live mode: refresh on a configurable interval (`-live -interval=30s`)
- Metric / imperial units, configurable via file or CLI

## Install

```sh
git clone https://github.com/TheOddKn1ght/wrep.git
cd wrep
go build
```

Or `go install github.com/TheOddKn1ght/wrep@latest`.

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
| `-apiprovider`  | `wttr.in` or `weatherapi` |
| `-f`            | Show an N-day forecast (e.g. `-f 3`). wttr.in caps at 3. |
| `-fancy`        | Color + emoji output |
| `-no-color`     | Disable color escapes (honors `NO_COLOR` env too) |
| `-json`         | Emit raw JSON instead of formatted output |
| `-v`            | Verbose (prints the request URL to stderr) |
| `-q`            | Quiet (suppresses warnings) |
| `-V`, `-version`| Print version and exit |
| `-config`       | Directory containing `.wrep` (default: `$HOME`) |
| `-live`         | Refresh on an interval until interrupted (Ctrl+C to exit) |
| `-interval`     | Refresh interval as a Go duration (e.g. `30s`, `5m`); default `60s`, min `5s` |

### Examples
```sh
./wrep -city=Berlin -fancy
./wrep -f 3 -fancy
./wrep -apiprovider=weatherapi -apikey=$KEY -city=Tokyo -unit=imperial
./wrep -json | jq '.temp_c'
```

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

On first run, a config file is created at `~/.wrep`:

```
apiKey=your_api_key_here
defaultCity=Moscow
units=metric
apiProvider=wttr.in
fancy=off
verbose=off
noColor=off
live=off
# interval=60s
```

Edit it to set your defaults. CLI flags override file values.

| Key | Values |
|-----|--------|
| `apiKey`      | Your WeatherAPI key (not required for wttr.in) |
| `defaultCity` | Default city |
| `units`       | `metric` or `imperial` |
| `apiProvider` | `wttr.in` or `weatherapi` |
| `fancy`       | `on` / `off` (also accepts `true`/`false`/`yes`/`1`) |
| `verbose`     | `on` / `off` |
| `noColor`     | `on` / `off` |
| `live`        | `on` / `off` — enable live refresh mode |
| `interval`    | Go duration string (e.g. `30s`, `5m`); min `5s` |

## Build a tagged release

```sh
go build -ldflags "-X main.version=$(git describe --tags --always)" -o wrep
```
