package main

import (
	"fmt"
	"io"
)

var weatherArtTable = map[WeatherType][]string{
	Sunny: {
		`    \   /    `,
		`     .-.     `,
		`  - (   ) -  `,
		`     '-'     `,
		`    /   \    `,
	},
	Cloudy: {
		`             `,
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
		`             `,
	},
	Rainy: {
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
		`  ' ' ' '    `,
		`   ' ' ' '   `,
	},
	Snowy: {
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
		`   *  *  *   `,
		`  *  *  *    `,
	},
	Stormy: {
		`     .--.    `,
		`  .-(    ).  `,
		` (___.__)__) `,
		`    /_  /_   `,
		`    /   /    `,
	},
	Foggy: {
		`             `,
		`  _ - _ - _  `,
		`   - _ - _ - `,
		`  _ - _ - _  `,
		`   - _ - _ - `,
	},
	Unknown: {
		`    .---.    `,
		`    |   |    `,
		`    | ? |    `,
		`    |   |    `,
		`    '---'    `,
	},
}

func WeatherArt(wt WeatherType) []string {
	if art, ok := weatherArtTable[wt]; ok {
		return art
	}
	return weatherArtTable[Unknown]
}

func renderArt(w io.Writer, info WeatherInfo, config Config) {
	color, reset, bold := "", "", ""
	if useColor(config) {
		color = WeatherColor(info.Type)
		reset = Reset
		bold = Bold
	}

	condition := info.Description
	if config.Fancy {
		condition = WeatherEmoji(info.Type) + " " + condition
	}

	infoLines := []struct{ label, value string }{
		{"City", config.City},
		{"Condition", condition},
		{"Temperature", formatTemp(info.TempC, info.TempF, config.Unit)},
		{"UV Index", formatUV(info.UVIndex)},
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
		artCell := padRight(artLine, artW)

		var infoCell string
		if i < len(infoLines) {
			infoCell = fmt.Sprintf("%s%s%s: %s", bold, infoLines[i].label, reset, infoLines[i].value)
		}

		fmt.Fprintf(w, "%s%s%s  %s\n", color, artCell, reset, infoCell)
	}
	fmt.Fprintln(w)
}
