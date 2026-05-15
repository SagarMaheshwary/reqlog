package scanner

var DefaultKeys = []string{
	"request_id",
	"req_id",
	"trace_id",
	"correlation_id",
}

var TimestampKeys = []string{
	"time",
	"timestamp",
	"ts",
}

var MessageKeys = map[string]struct{}{
	"msg":     {},
	"message": {},
	"error":   {},
}
