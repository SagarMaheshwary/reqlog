package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/parser"
)

func createPlainLogFile(b *testing.B, dir, name string, lines int, matchEvery int) string {
	b.Helper()

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		b.Fatalf("failed to create log file: %v", err)
	}
	defer f.Close()

	baseTime := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	var sb strings.Builder

	for i := 0; i < lines; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Second).Format(time.RFC3339)
		reqID := "other-id"
		if matchEvery > 0 && i%matchEvery == 0 {
			reqID = "abc123"
		}
		fmt.Fprintf(&sb, "%s request_id=%s log message number %d\n", ts, reqID, i)
	}

	if _, err := f.WriteString(sb.String()); err != nil {
		b.Fatalf("failed to write log file: %v", err)
	}

	return path
}

func createJSONLogFile(b *testing.B, dir, name string, lines int, matchEvery int) string {
	b.Helper()

	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		b.Fatalf("failed to create json log file: %v", err)
	}
	defer f.Close()

	baseTime := time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC)
	var sb strings.Builder

	for i := 0; i < lines; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Second).Format(time.RFC3339)
		reqID := "other-id"
		if matchEvery > 0 && i%matchEvery == 0 {
			reqID = "json-abc"
		}
		fmt.Fprintf(&sb, `{"time":"%s","request_id":"%s","message":"log message number %d"}`+"\n", ts, reqID, i)
	}

	if _, err := f.WriteString(sb.String()); err != nil {
		b.Fatalf("failed to write json log file: %v", err)
	}

	return path
}

func BenchmarkScanDir_PlainText(b *testing.B) {
	dir := b.TempDir()
	createPlainLogFile(b, dir, "api.log", 200_000, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p, _ := parser.NewParser(parser.TypeText)

		scn := NewFileScanner(p)
		entries, err := scn.Scan(ScanConfig{
			Dir:         dir,
			SearchValue: "abc123",
			IgnoreCase:  false,
			Key:         "",
			Since:       "",
		})
		if err != nil {
			b.Fatalf("scan failed: %v", err)
		}
		if len(entries) == 0 {
			b.Fatalf("expected matches, got none")
		}
	}
}

func BenchmarkScanDir_JSON(b *testing.B) {
	dir := b.TempDir()
	createJSONLogFile(b, dir, "api.log", 200_000, 1000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p, _ := parser.NewParser(parser.TypeJSON)

		scn := NewFileScanner(p)
		entries, err := scn.Scan(ScanConfig{
			Dir:         dir,
			SearchValue: "json-abc",
			IgnoreCase:  false,
			Key:         "",
			Since:       "",
		})
		if err != nil {
			b.Fatalf("scan failed: %v", err)
		}
		if len(entries) == 0 {
			b.Fatalf("expected matches, got none")
		}
	}
}
