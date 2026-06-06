package transport

import "testing"

func TestBuildServiceMatcher(t *testing.T) {
	tests := []struct {
		name     string
		services []string
		file     string
		want     bool
	}{
		{"no filters matches all", nil, "api.log", true},
		{"exact match + empty service", []string{"api", ""}, "api.log", true},
		{"exact no match", []string{"db"}, "api.log", false},
		{"prefix wildcard", []string{"api-*"}, "api-prod.log", true},
		{"prefix wildcard no match", []string{"api-*"}, "worker.log", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := buildServiceMatcher(tt.services)

			got := match(tt.file)

			if got != tt.want {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}

func TestIsLogFile(t *testing.T) {
	tests := []struct {
		name string
		file string
		want bool
	}{
		{"log file", "app.log", true},
		{"text file", "app.txt", false},
		{"no extension", "app", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLogFile(tt.file)

			if got != tt.want {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}

func TestBuildServiceMatcher_MixedExactAndWildcard(t *testing.T) {
	match := buildServiceMatcher([]string{"api-*", "worker"})

	tests := []struct {
		name string
		file string
		want bool
	}{
		{
			name: "matches wildcard prefix",
			file: "api-prod.log",
			want: true,
		},
		{
			name: "matches exact service",
			file: "worker.log",
			want: true,
		},
		{
			name: "does not match non matching service",
			file: "db.log",
			want: false,
		},
		{
			name: "does not match partial exact service",
			file: "worker-prod.log",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := match(tt.file)

			if got != tt.want {
				t.Fatalf("match(%q): expected %v got %v", tt.file, tt.want, got)
			}
		})
	}
}

func TestQuoteShellArg(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "empty string",
			in:   "",
			want: "''",
		},
		{
			name: "plain string",
			in:   "hello",
			want: "'hello'",
		},
		{
			name: "contains spaces",
			in:   "hello world",
			want: "'hello world'",
		},
		{
			name: "single quote",
			in:   "abc'def",
			want: `'abc'"'"'def'`,
		},
		{
			name: "multiple single quotes",
			in:   "a'b'c",
			want: `'a'"'"'b'"'"'c'`,
		},
		{
			name: "already contains double quotes",
			in:   `hello "world"`,
			want: `'hello "world"'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteShellArg(tt.in)

			if got != tt.want {
				t.Fatalf("expected %q got %q", tt.want, got)
			}
		})
	}
}
