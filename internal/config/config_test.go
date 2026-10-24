package config

import "testing"

func TestConfig_LiveOnly(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want bool
	}{
		{
			name: "follow without search value",
			cfg:  Config{Follow: true},
			want: true,
		},
		{
			name: "follow with search value",
			cfg:  Config{Follow: true, SearchValue: "abc123"},
			want: false,
		},
		{
			name: "search value without follow",
			cfg:  Config{SearchValue: "abc123"},
			want: false,
		},
		{
			name: "neither follow nor search value",
			cfg:  Config{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.LiveOnly(); got != tt.want {
				t.Fatalf("expected %v got %v", tt.want, got)
			}
		})
	}
}
