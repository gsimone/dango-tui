package domain

import (
	"math"
	"math/rand/v2"
	"sync"
)

type OKLCH [3]float64

func clamp(value float64, bounds ...float64) float64 {
	lower, upper := 0.0, 1.0
	if len(bounds) >= 1 {
		lower = bounds[0]
	}
	if len(bounds) >= 2 {
		upper = bounds[1]
	}
	return math.Min(upper, math.Max(lower, value))
}

// OKLCHToSRGB converts an authored OKLCH token to a terminal-safe RGB triple.
func OKLCHToSRGB(token OKLCH) [3]int {
	lightness, chroma, hue := token[0], token[1], token[2]
	radians := (hue * math.Pi) / 180
	a := chroma * math.Cos(radians)
	b := chroma * math.Sin(radians)

	l := lightness + 0.3963377774*a + 0.2158037573*b
	m := lightness - 0.1055613458*a - 0.0638541728*b
	s := lightness - 0.0894841775*a - 1.291485548*b
	l3 := l * l * l
	m3 := m * m * m
	s3 := s * s * s

	linear := [3]float64{
		4.0767416621*l3 - 3.3077115913*m3 + 0.2309699292*s3,
		-1.2684380046*l3 + 2.6097574011*m3 - 0.3413193965*s3,
		-0.0041960863*l3 - 0.7034186147*m3 + 1.707614701*s3,
	}

	gamma := func(channel float64) float64 {
		clipped := clamp(channel)
		if clipped <= 0.0031308 {
			return 12.92 * clipped
		}
		return 1.055*math.Pow(clipped, 1/2.4) - 0.055
	}

	return [3]int{
		int(math.Round(clamp(gamma(linear[0])) * 255)),
		int(math.Round(clamp(gamma(linear[1])) * 255)),
		int(math.Round(clamp(gamma(linear[2])) * 255)),
	}
}

func OKLCHToHex(token OKLCH) string {
	rgb := OKLCHToSRGB(token)
	return sprintfHex(rgb[0], rgb[1], rgb[2])
}

func sprintfHex(r, g, b int) string {
	const hexdigits = "0123456789abcdef"
	out := [7]byte{'#', 0, 0, 0, 0, 0, 0}
	out[1] = hexdigits[r>>4]
	out[2] = hexdigits[r&0x0f]
	out[3] = hexdigits[g>>4]
	out[4] = hexdigits[g&0x0f]
	out[5] = hexdigits[b>>4]
	out[6] = hexdigits[b&0x0f]
	return string(out[:])
}

// Status colors start life as an OKLCH triple. Chrome is hex-locked below.
var OKLCHTokens = map[string]OKLCH{
	"surface":       {0.18, 0.025, 260},
	"surfaceRaised": {0.235, 0.03, 260},
	"border":        {0.37, 0.035, 260},
	"text":          {0.93, 0.018, 260},
	"muted":         {0.66, 0.026, 260},
	"focus":         {0.76, 0.13, 255},
	"draft":         {0.62, 0.025, 260},
	"open":          {0.67, 0.034, 260},
	"queued":        {0.8, 0.155, 88},
	"ciFailure":     {0.66, 0.19, 35},
	"reviewBlocked": {0.6, 0.22, 19},
	"ready":         {0.72, 0.16, 145},
	"merged":        {0.65, 0.145, 305},
	"success":       {0.76, 0.13, 145},
	"warning":       {0.8, 0.14, 88},
	"logoRed":       {0.63, 0.22, 25},
	"logoOrange":    {0.72, 0.18, 55},
	"logoYellow":    {0.85, 0.16, 95},
	"logoGreen":     {0.72, 0.18, 145},
	"logoBlue":      {0.62, 0.16, 250},
	"logoPurple":    {0.60, 0.20, 310},
	"logoPink":      {0.70, 0.18, 350},
}

// LogoTokens is the seven-ink palette for the header o-o-o mark only.
var LogoTokens = []string{
	"logoRed", "logoOrange", "logoYellow", "logoGreen", "logoBlue", "logoPurple", "logoPink",
}

// ChromeHex locks the field, selected row, type, and connectors to the mark.
// Type is paper or meta. Status words keep their OKLCH hues.
var ChromeHex = map[string]string{
	"surface":       "#14120f",
	"surfaceRaised": "#242018",
	"paper":         "#f2ebe0",
	"text":          "#f2ebe0",
	"meta":          "#9a8f82",
	"muted":         "#9a8f82",
	"focus":         "#f2ebe0",
	"stick":         "#8f8678",
	"border":        "#8f8678",
}

var TerminalColors = func() map[string]string {
	out := make(map[string]string, len(OKLCHTokens)+len(ChromeHex))
	for name, token := range OKLCHTokens {
		out[name] = OKLCHToHex(token)
	}
	for name, hex := range ChromeHex {
		out[name] = hex
	}
	return out
}()

func Color(token string) string {
	if hex, ok := TerminalColors[token]; ok {
		return hex
	}
	return "#ffffff"
}

func IsLogoToken(name string) bool {
	for _, token := range LogoTokens {
		if token == name {
			return true
		}
	}
	return false
}

func PickLogoDots(intn func(int) int) [3]string {
	if intn == nil {
		intn = rand.IntN
	}
	pool := append([]string(nil), LogoTokens...)
	var out [3]string
	for i := 0; i < 3; i++ {
		j := intn(len(pool))
		if j < 0 {
			j = 0
		}
		if j >= len(pool) {
			j = len(pool) - 1
		}
		out[i] = pool[j]
		pool = append(pool[:j], pool[j+1:]...)
	}
	return out
}

var (
	processLogo     [3]string
	processLogoOnce sync.Once
)

func ProcessLogoDots() [3]string {
	processLogoOnce.Do(func() {
		processLogo = PickLogoDots(rand.IntN)
	})
	return processLogo
}
