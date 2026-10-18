package config

type OutputFormat string

const (
	OutputPretty OutputFormat = "pretty"
	OutputJSON   OutputFormat = "json"
)

type LogFormat string

const (
	FormatAuto LogFormat = "auto"
	FormatJSON LogFormat = "json"
	FormatText LogFormat = "text"
)

type Source string

const (
	SourceFile   Source = "file"
	SourceDocker Source = "docker"
)

type Config struct {
	Version     bool
	Dir         string
	SearchValue string
	IgnoreCase  bool
	Keys        []string
	Since       string
	Limit       int
	Recursive   bool
	Services    []string
	Latest      bool
	Context     int
	Fields      []string
	Verbose     bool

	Source Source
	Follow bool

	Output OutputFormat
	Format LogFormat

	Host   string
	Config *SSH
}

// LiveOnly reports whether reqlog should stream every newly appended log line
// instead of searching: --follow is set and no search value was provided.
func (c *Config) LiveOnly() bool {
	return c.Follow && c.SearchValue == ""
}

var DefaultKeys = []string{
	"request_id",
	"req_id",
	"trace_id",
	"correlation_id",
}
