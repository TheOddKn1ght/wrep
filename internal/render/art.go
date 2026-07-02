package render

import (
	"fmt"
	"io"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

var weatherArtTable = map[weather.WeatherType][]string{
	weather.Sunny: {
		`    \   /    `,
		`     .-.     `,
		`  - (   ) -  `,
		`     '-'     `,
		`    /   \    `,
	},
	weather.Cloudy: {
		`             `,
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
		`             `,
	},
	weather.Rainy: {
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
		`  ' ' ' '    `,
		`   ' ' ' '   `,
	},
	weather.Snowy: {
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
		`   *  *  *   `,
		`  *  *  *    `,
	},
	weather.Stormy: {
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
		`    /_  /_   `,
		`    /   /    `,
	},
	weather.Foggy: {
		`             `,
		`  _ - _ - _  `,
		`   - _ - _ - `,
		`  _ - _ - _  `,
		`   - _ - _ - `,
	},
	weather.Unknown: {
		`    .---.    `,
		`    |   |    `,
		`    | ? |    `,
		`    |   |    `,
		`    '---'    `,
	},
}

func WeatherArt(wt weather.WeatherType) []string {
	if art, ok := weatherArtTable[wt]; ok {
		return art
	}
	return weatherArtTable[weather.Unknown]
}

func renderArt(w io.Writer, info weather.WeatherInfo, cfg config.Config) {
	color, reset, bold := "", "", ""
	if useColor(cfg) {
		color = WeatherColor(info.Type)
		reset = Reset
		bold = Bold
	}

	condition := info.Description
	if cfg.Fancy {
		condition = WeatherEmoji(info.Type) + " " + condition
	}

	infoLines := []struct{ label, value string }{
		{"City", cfg.City},
		{"Condition", condition},
		{"Temperature", FormatTemp(info.TempC, info.TempF, cfg.Unit)},
		{"Feels like", FormatTemp(info.FeelsLikeC, info.FeelsLikeF, cfg.Unit)},
		{"Humidity", fmt.Sprintf("%.0f%%", info.Humidity)},
		{"Wind", FormatWind(info.WindKph, info.WindMph, cfg.Unit) + " " + weather.WindCompass(info.WindDegree)},
		{"UV Index", formatUV(info.UVIndex)},
	}
	if info.Sunrise != "" && info.Sunset != "" {
		infoLines = append(infoLines,
			struct{ label, value string }{"Sunrise", info.Sunrise},
			struct{ label, value string }{"Sunset", info.Sunset},
		)
	}

	art := WeatherArt(info.Type)

	artW := 0
	for _, line := range art {
		if n := len([]rune(line)); n > artW {
			artW = n
		}
	}

	rows := len(art)
	if len(infoLines) > rows {
		rows = len(infoLines)
	}

	fmt.Fprintln(w)
	for i := 0; i < rows; i++ {
		var artLine string
		if i < len(art) {
			artLine = art[i]
		}
		artCell := PadRight(artLine, artW)

		var infoCell string
		if i < len(infoLines) {
			infoCell = fmt.Sprintf("%s%s%s: %s", bold, infoLines[i].label, reset, infoLines[i].value)
		}

		fmt.Fprintf(w, "%s%s%s  %s\n", color, artCell, reset, infoCell)
	}
	fmt.Fprintln(w)
}
