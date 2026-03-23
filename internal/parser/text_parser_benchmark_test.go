package parser

import (
	"testing"
)

var sampleLine = "2024-03-10T12:00:00Z user=123 status=ok method=GET path=/api/v1 msg=hello_world"

var sampleLineWithQuotes = `2024-03-10T12:00:00Z user="123" status="ok" method=GET path=/api msg="hello world"`

func BenchmarkParse(b *testing.B) {
	p := TextParser{}

	for i := 0; i < b.N; i++ {
		_, _ = p.Parse(sampleLine, "svc")
	}
}

func BenchmarkExtractField(b *testing.B) {
	p := TextParser{}

	keys := []string{"status", "user"}

	for i := 0; i < b.N; i++ {
		_, _ = p.ExtractField(sampleLine, keys)
	}
}

func BenchmarkExtractField_WithQuotes_NoHandling(b *testing.B) {
	p := TextParser{}

	for i := 0; i < b.N; i++ {
		_, _ = p.ExtractField(sampleLineWithQuotes, []string{"status"})
	}
}
