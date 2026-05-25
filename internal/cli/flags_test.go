package cli

import (
	"reflect"
	"testing"

	"github.com/sagarmaheshwary/reqlog/internal/config"
)

func TestIsTailStyleLimit(t *testing.T) {
	tests := []struct {
		arg  string
		want bool
	}{
		{"-10", true},
		{"-1", true},
		{"-999", true},

		{"10", false},
		{"--10", false},
		{"-n", false},
		{"-abc", false},
		{"-1a", false},
		{"-", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			got := isTailStyleLimit(tt.arg)

			if got != tt.want {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}

func TestNormalizeArgs(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "tail style limit",
			in:   []string{"-10"},
			want: []string{"-n", "10"},
		},
		{
			name: "mixed args",
			in:   []string{"--follow", "-20", "abc123"},
			want: []string{"--follow", "-n", "20", "abc123"},
		},
		{
			name: "normal args unchanged",
			in:   []string{"--limit", "10"},
			want: []string{"--limit", "10"},
		},
		{
			name: "non numeric dash arg unchanged",
			in:   []string{"-abc"},
			want: []string{"-abc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeArgs(tt.in)

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}

func TestApplyDerivedConfig(t *testing.T) {
	t.Run("applies derived values", func(t *testing.T) {
		cfg := &config.Config{}

		opts := &flagOptions{
			key:     "trace_id",
			service: "auth,db",
			source:  "docker",
			output:  "json",
			format:  "text",
		}

		err := applyDerivedConfig(cfg, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(cfg.Keys, []string{"trace_id"}) {
			t.Fatalf("keys: expected %v got %v", []string{"trace_id"}, cfg.Keys)
		}

		if !reflect.DeepEqual(cfg.Services, []string{"auth", "db"}) {
			t.Fatalf("services: expected %v got %v", []string{"auth", "db"}, cfg.Services)
		}

		if cfg.Source != config.SourceDocker {
			t.Fatalf("source: expected %v got %v", config.SourceDocker, cfg.Source)
		}

		if cfg.Output != config.OutputJSON {
			t.Fatalf("output: expected %v got %v", config.OutputJSON, cfg.Output)
		}

		if cfg.Format != config.FormatText {
			t.Fatalf("format: expected %v got %v", config.FormatText, cfg.Format)
		}
	})

	t.Run("uses default keys when key not provided", func(t *testing.T) {
		cfg := &config.Config{}
		opts := &flagOptions{}

		err := applyDerivedConfig(cfg, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !reflect.DeepEqual(cfg.Keys, config.DefaultKeys) {
			t.Fatalf("keys: expected %v got %v", config.DefaultKeys, cfg.Keys)
		}
	})

	t.Run("returns ssh config error when host is set", func(t *testing.T) {
		cfg := &config.Config{
			Host: "prod",
		}

		opts := &flagOptions{
			config: "/does/not/exist.yaml",
		}

		err := applyDerivedConfig(cfg, opts)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name: "valid pretty output",
			cfg: &config.Config{
				Output: config.OutputPretty,
				Format: config.FormatAuto,
			},
			wantErr: false,
		},
		{
			name: "valid json output",
			cfg: &config.Config{
				Output: config.OutputJSON,
				Format: config.FormatJSON,
			},
			wantErr: false,
		},
		{
			name: "invalid output",
			cfg: &config.Config{
				Output: "xml",
				Format: config.FormatAuto,
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			cfg: &config.Config{
				Output: config.OutputPretty,
				Format: "yaml",
			},
			wantErr: true,
		},
		{
			name: "latest without limit",
			cfg: &config.Config{
				Output: config.OutputPretty,
				Format: config.FormatAuto,
				Latest: true,
				Limit:  0,
			},
			wantErr: true,
		},
		{
			name: "latest with limit",
			cfg: &config.Config{
				Output: config.OutputPretty,
				Format: config.FormatAuto,
				Latest: true,
				Limit:  10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfig(tt.cfg)

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%v got err=%v", tt.wantErr, err)
			}
		})
	}
}
