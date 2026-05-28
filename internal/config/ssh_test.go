package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:  "explicit path returned as-is",
			input: "/tmp/custom.yaml",
		},
		{
			name:  "empty path uses default",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetConfigPath(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.input != "" && got != tt.input {
				t.Fatalf("expected %q got %q", tt.input, got)
			}

			if tt.input == "" && got == "" {
				t.Fatal("expected default path, got empty")
			}
		})
	}
}

func TestNewSSH(t *testing.T) {
	validYAML := `
defaults:
  key: "default-key"
hosts:
  prod:
    host: "1.1.1.1"
    user: "root"
`

	tests := []struct {
		name        string
		setup       func(t *testing.T) string // returns path
		wantErr     bool
		errContains string
		assert      func(t *testing.T, cfg *SSH)
	}{
		{
			name: "valid explicit path",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "config.yaml")
				if err := os.WriteFile(path, []byte(validYAML), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			assert: func(t *testing.T, cfg *SSH) {
				if cfg.Hosts["prod"].User != "root" {
					t.Fatalf("unexpected user: %v", cfg.Hosts["prod"].User)
				}
			},
		},
		{
			name: "file not found",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "missing.yaml")
			},
			wantErr:     true,
			errContains: "SSH config not found",
		},
		{
			name: "invalid yaml parse error",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				path := filepath.Join(dir, "bad.yaml")
				if err := os.WriteFile(path, []byte("invalid: [yaml"), 0o644); err != nil {
					t.Fatal(err)
				}
				return path
			},
			wantErr:     true,
			errContains: "parse ssh config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)

			cfg, err := NewSSH(path)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q got %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.assert != nil {
				tt.assert(t, cfg)
			}
		})
	}
}

func TestParseSSH(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		assert      func(t *testing.T, cfg *SSH)
		errContains string
	}{
		{
			name: "valid config with defaults applied",
			input: `
defaults:
  key: "default-key"
  timeout: 15s
hosts:
  prod:
    host: "10.0.0.1"
    user: "root"
`,
			assert: func(t *testing.T, cfg *SSH) {
				h := cfg.Hosts["prod"]

				if h.Port != sshDefaultPort {
					t.Fatalf("expected default port %d got %d", sshDefaultPort, h.Port)
				}
				if h.Key != "default-key" {
					t.Fatalf("expected default key got %q", h.Key)
				}
				if h.Timeout != 15*time.Second {
					t.Fatalf("expected timeout 15s got %v", h.Timeout)
				}
			},
		},
		{
			name: "host overrides defaults",
			input: `
defaults:
  key: "default-key"
  timeout: 10s
hosts:
  prod:
    host: "10.0.0.1"
    user: "root"
    key: "host-key"
    port: 2222
    timeout: 5s
`,
			assert: func(t *testing.T, cfg *SSH) {
				h := cfg.Hosts["prod"]

				if h.Key != "host-key" {
					t.Fatalf("expected host key override got %q", h.Key)
				}
				if h.Port != 2222 {
					t.Fatalf("expected port 2222 got %d", h.Port)
				}
				if h.Timeout != 5*time.Second {
					t.Fatalf("expected timeout override got %v", h.Timeout)
				}
			},
		},
		{
			name: "missing hosts should fail",
			input: `
defaults:
  key: "x"
`,
			wantErr:     true,
			errContains: "no hosts defined",
		},
		{
			name: "missing host field",
			input: `
hosts:
  prod:
    user: root
    key: x
`,
			wantErr:     true,
			errContains: "missing host",
		},
		{
			name: "missing user field",
			input: `
hosts:
  prod:
    host: 1.1.1.1
    key: x
`,
			wantErr:     true,
			errContains: "missing user",
		},
		{
			name: "missing key field uses default then validates",
			input: `
defaults:
  key: "default-key"
hosts:
  prod:
    host: 1.1.1.1
    user: root
`,
			assert: func(t *testing.T, cfg *SSH) {
				h := cfg.Hosts["prod"]
				if h.Key != "default-key" {
					t.Fatalf("expected default key got %q", h.Key)
				}
			},
		},
		{
			name: "invalid yaml",
			input: `
hosts:
  prod:
    host: [invalid`,
			wantErr:     true,
			errContains: "unmarshal",
		},
		{
			name: "timeout fallback default when missing everywhere",
			input: `
hosts:
  prod:
    host: 1.1.1.1
    user: root
    key: x
`,
			assert: func(t *testing.T, cfg *SSH) {
				h := cfg.Hosts["prod"]
				if h.Timeout != sshDefaultTimeout {
					t.Fatalf("expected default timeout %v got %v", sshDefaultTimeout, h.Timeout)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := ParseSSH([]byte(tt.input))

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error but got nil")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q got %v", tt.errContains, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.assert != nil {
				tt.assert(t, cfg)
			}
		})
	}
}
