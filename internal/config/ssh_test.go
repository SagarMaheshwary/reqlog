package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestParseSSH(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *SSH
		wantErr string
	}{
		{
			name: "valid config with defaults applied",
			input: `
defaults:
  key: ~/.ssh/id_rsa

hosts:
  prod:
    host: prod.example.com
    user: ubuntu
`,
			want: &SSH{
				Defaults: Defaults{
					Key: "~/.ssh/id_rsa",
				},
				Hosts: map[string]Host{
					"prod": {
						Host: "prod.example.com",
						User: "ubuntu",
						Port: sshDefaultPort,
						Key:  "~/.ssh/id_rsa",
					},
				},
			},
		},
		{
			name: "explicit values override defaults",
			input: `
defaults:
  key: ~/.ssh/id_rsa

hosts:
  prod:
    host: prod.example.com
    user: ubuntu
    port: 2222
    key: ~/.ssh/custom_key
`,
			want: &SSH{
				Defaults: Defaults{
					Key: "~/.ssh/id_rsa",
				},
				Hosts: map[string]Host{
					"prod": {
						Host: "prod.example.com",
						User: "ubuntu",
						Port: 2222,
						Key:  "~/.ssh/custom_key",
					},
				},
			},
		},
		{
			name: "invalid yaml",
			input: `
hosts: [:
`,
			wantErr: "yaml:",
		},
		{
			name: "missing host validation error",
			input: `
hosts:
  prod:
    user: ubuntu
    key: ~/.ssh/id_rsa
`,
			wantErr: `host "prod": missing host`,
		},
		{
			name: "no hosts",
			input: `
defaults:
  key: ~/.ssh/id_rsa
`,
			wantErr: "ssh config: no hosts defined",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSSH([]byte(tt.input))

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q got nil", tt.wantErr)
				}

				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q got %q", tt.wantErr, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(tt.want, got) {
				t.Fatalf("ParseSSH() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewSSH(t *testing.T) {
	t.Run("reads config from file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")

		data := `
hosts:
  prod:
    host: prod.example.com
    user: ubuntu
    key: ~/.ssh/id_rsa
`

		if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg, err := NewSSH(path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Hosts["prod"].Host != "prod.example.com" {
			t.Fatalf("unexpected host: %+v", cfg.Hosts["prod"])
		}
	})

	t.Run("missing file returns helpful error", func(t *testing.T) {
		_, err := NewSSH("/does/not/exist.yaml")

		if err == nil {
			t.Fatal("expected error")
		}

		if !strings.Contains(err.Error(), "SSH config not found") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
