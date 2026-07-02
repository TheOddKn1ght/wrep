package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

// renderFormat prints a single line built from a user template — intended for
// status bars (tmux, i3, polybar), so no colors and no decoration.
func renderFormat(w io.Writer, info weather.WeatherInfo, cfg config.Config) {
	fmt.Fprintln(w, ExpandFormat(cfg.Format, info, cfg))
}

// ExpandFormat substitutes {placeholder} tokens. Unknown tokens are left
// verbatim so typos are visible rather than silently swallowed.
func ExpandFormat(format string, info weather.WeatherInfo, cfg config.Config) string {
	minTemp, maxTemp := "", ""
	if len(info.Forecast) > 0 {
		d := info.Forecast[0]
		minTemp = FormatTemp(d.MinTempC, d.MinTempF, cfg.Unit)
		maxTemp = FormatTemp(d.MaxTempC, d.MaxTempF, cfg.Unit)
	}
	r := strings.NewReplacer(
		"{city}", cfg.City,
		"{desc}", info.Description,
		"{emoji}", WeatherEmoji(info.Type),
		"{temp}", FormatTemp(info.TempC, info.TempF, cfg.Unit),
		"{feels}", FormatTemp(info.FeelsLikeC, info.FeelsLikeF, cfg.Unit),
		"{tempc}", fmt.Sprintf("%.1f", info.TempC),
		"{tempf}", fmt.Sprintf("%.1f", info.TempF),
		"{humidity}", fmt.Sprintf("%.0f", info.Humidity),
		"{wind}", FormatWind(info.WindKph, info.WindMph, cfg.Unit),
		"{winddir}", weather.WindCompass(info.WindDegree),
		"{uv}", formatUV(info.UVIndex),
		"{sunrise}", info.Sunrise,
		"{sunset}", info.Sunset,
		"{min}", minTemp,
		"{max}", maxTemp,
	)
	return r.Replace(format)
}
