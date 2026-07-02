package render

import (
	"fmt"
	"io"
	"os"

	"github.com/TheOddKn1ght/wrep/internal/config"
	"github.com/TheOddKn1ght/wrep/internal/weather"
)

var sparkLevels = []rune("▁▂▃▄▅▆▇█")

// Sparkline maps values onto block-element runes scaled between the slice's
// min and max. A flat series renders at mid height.
func Sparkline(vals []float64) string {
	if len(vals) == 0 {
		return ""
	}
	lo, hi := vals[0], vals[0]
	for _, v := range vals {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	span := hi - lo
	out := make([]rune, len(vals))
	for i, v := range vals {
		idx := len(sparkLevels) / 2
		if span > 1e-9 {
			idx = int((v - lo) / span * float64(len(sparkLevels)-1))
		}
		out[i] = sparkLevels[idx]
	}
	return string(out)
}

func renderHourly(w io.Writer, info weather.WeatherInfo, cfg config.Config) {
	if len(info.Hourly) == 0 {
		if !cfg.Quiet {
			fmt.Fprintln(os.Stderr, "wrep: no hourly data available")
		}
		return
	}
	color := useColor(cfg)

	fmt.Fprintln(w)
	header := "Next 24 hours"
	if cfg.Fancy {
		header = "🕐 " + header
	}
	if color {
		header = Bold + header + Reset
	}
	fmt.Fprintln(w, header)

	temps := make([]float64, len(info.Hourly))
	loIdx, hiIdx := 0, 0
	for i, p := range info.Hourly {
		t := p.TempC
		if cfg.Unit == config.UnitImperial {
			t = p.TempF
		}
		temps[i] = t
		if t < temps[loIdx] {
			loIdx = i
		}
		if t > temps[hiIdx] {
			hiIdx = i
		}
	}
	lo := info.Hourly[loIdx]
	hi := info.Hourly[hiIdx]
	fmt.Fprintf(w, "  %s  %s – %s\n\n",
		Sparkline(temps),
		FormatTemp(lo.TempC, lo.TempF, cfg.Unit),
		FormatTemp(hi.TempC, hi.TempF, cfg.Unit))

	// Sample down to ~9 rows so hourly providers don't print 25 lines.
	step := 1
	if len(info.Hourly) > 9 {
		step = (len(info.Hourly) + 8) / 9
	}
	for i := 0; i < len(info.Hourly); i += step {
		p := info.Hourly[i]
		line := fmt.Sprintf("  %s  %7s  %s",
			p.Time.Format("15:04"),
			FormatTemp(p.TempC, p.TempF, cfg.Unit),
			p.Description)
		if color {
			line = WeatherColor(p.Type) + line + Reset
		}
		fmt.Fprintln(w, line)
	}
	fmt.Fprintln(w)
}
