 # 🚧 Work in Progress 🚧
 
 # wrep

A simple command-line weather reporter written in Go. It fetches and displays current weather information for a specified city using either [wttr.in](https://wttr.in) or [WeatherAPI](https://www.weatherapi.com/).

## Features
- Fetches weather data from two providers: wttr.in (default) and WeatherAPI
- Displays temperature, weather description, and UV index
- Supports metric and imperial units
- Configurable via a config file and command-line flags

## Installation

1. **Clone the repository:**
```sh
git clone https://github.com/TheOddKn1ght/wrep.git
cd wrep
```
2. **Build the project:**
```sh
go build
```

## Usage

Run the program from the command line:
```sh
./wrep [flags]
```

### Command-line Flags
- `-city`         Override city from CLI (e.g., `-city=London`)
- `-unit`         Override unit from CLI (`metric` or `imperial`)
- `-apikey`       Override API key from CLI (for WeatherAPI)
- `-apiprovider`  API provider to use: `wttr.in` or `weatherapi`
- `-v`            Verbose output

### Example
```sh
./wrep -city=Berlin -unit=imperial -apiprovider=weatherapi -apikey=YOUR_API_KEY
```

## Configuration

On first run, a config file will be created at `~/.wrep` with default values:
```
apiKey=your_api_key_here
defaultCity=Moscow
units=metric
apiProvider=wttr.in
```
You can edit this file to set your preferred defaults. Command-line flags override config file values.

- `apiKey`      Your WeatherAPI key (not required for wttr.in)
- `defaultCity` Default city to fetch weather for
- `units`       `metric` or `imperial`
- `apiProvider` `wttr.in` or `weatherapi`
