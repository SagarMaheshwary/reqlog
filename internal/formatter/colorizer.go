package formatter

import "hash/fnv"

var colors = []string{
	"\033[38;5;67m", // muted blue
	"\033[38;5;68m", // steel blue
	"\033[38;5;69m", // soft blue

	"\033[38;5;108m", // desaturated cyan-green
	"\033[38;5;109m", // soft cyan
	"\033[38;5;110m", // teal

	"\033[38;5;142m", // olive green
	"\033[38;5;143m", // moss green
	"\033[38;5;114m", // soft green

	"\033[38;5;179m", // sand
	"\033[38;5;180m", // beige
	"\033[38;5;181m", // light tan

	"\033[38;5;173m", // muted orange
	"\033[38;5;174m", // soft orange

	"\033[38;5;139m", // dusty purple
	"\033[38;5;140m", // lavender
	"\033[38;5;141m", // soft violet

	"\033[38;5;175m", // pink-purple
	"\033[38;5;176m", // magenta soft

	"\033[38;5;166m", // warm orange-brown
}

const (
	reset = "\033[0m"
	dim   = "\033[2m"

	tsColor  = "\033[38;5;245m" // gray
	msgColor = "\033[38;5;252m" // near white
)

type Colorizer struct{}

func NewColorizer() *Colorizer {
	return &Colorizer{}
}

func (c *Colorizer) Color(service string) string {
	h := fnv32a(service)

	// bit mixing (important)
	h ^= h >> 16
	h *= 0x7feb352d
	h ^= h >> 15
	h *= 0x846ca68b
	h ^= h >> 16

	return colors[int(h)%len(colors)]
}

func fnv32a(text string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(text))
	return h.Sum32()
}
