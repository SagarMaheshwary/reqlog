package scanner

import (
	"reflect"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
)

func TestContextEngine_Handle(t *testing.T) {
	tests := []struct {
		name             string
		contextSize      int
		cfg              *Config
		inputs           []ContextLine
		expectedFields   []map[string]any
		expectedContext  []bool
		expectedContinue []bool
	}{
		{
			name:        "no context mode only collects matches",
			contextSize: 0,
			cfg: &Config{
				Limit: 10,
			},
			inputs: []ContextLine{
				{
					Line:    "2024-03-10T12:00:00Z user=ignore",
					Service: "svc",
					IsMatch: false,
				},
				{
					Line:    "2024-03-10T12:00:01Z user=123",
					Service: "svc",
					IsMatch: true,
					Entry: &domain.LogEntry{
						Timestamp: mustParseRFC3339("2024-03-10T12:00:01Z"),
						Service:   "svc",
						Fields:    map[string]any{"user": "123"},
					},
				},
			},
			expectedFields: []map[string]any{
				{"user": "123"},
			},
			expectedContext: []bool{
				false,
			},
			expectedContinue: []bool{
				true,
				true,
			},
		},
		{
			name:        "before and after context",
			contextSize: 1,
			cfg: &Config{
				Limit:       10,
				Keys:        []string{"user"},
				Context:     1,
				SearchValue: "123",
			},
			inputs: []ContextLine{
				{
					Line:    "2024-03-10T12:00:00Z user=before",
					Service: "svc",
					IsMatch: false,
				},
				{
					Line:    "2024-03-10T12:00:01Z user=123",
					Service: "svc",
					IsMatch: true,
					Entry: &domain.LogEntry{
						Timestamp: mustParseRFC3339("2024-03-10T12:00:01Z"),
						Service:   "svc",
						Fields:    map[string]any{"user": "123"},
					},
				},
				{
					Line:    "2024-03-10T12:00:02Z user=after",
					Service: "svc",
					IsMatch: false,
				},
			},
			expectedFields: []map[string]any{
				{"user": "before"},
				{"user": "123"},
				{"user": "after"},
			},
			expectedContext: []bool{
				true,
				false,
				true,
			},
			expectedContinue: []bool{
				true,
				true,
				true,
			},
		},
		{
			name:        "invalid context line is skipped",
			contextSize: 1,
			cfg: &Config{
				Limit:   10,
				Keys:    []string{"user"},
				Context: 1,
			},
			inputs: []ContextLine{
				{
					Line:    "invalid log line",
					Service: "svc",
					IsMatch: false,
				},
			},
			expectedFields:  nil,
			expectedContext: nil,
			expectedContinue: []bool{
				true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lp := NewLineProcessor(tt.cfg, NewTimeParser())

			collector, err := NewEntryCollector(tt.cfg, time.Now())
			if err != nil {
				t.Fatal(err)
			}

			engine := NewContextEngine(
				lp,
				collector,
				tt.contextSize,
			)

			var continues []bool

			for _, in := range tt.inputs {
				continues = append(continues, engine.Handle(in))
			}

			results := collector.Results()

			if !reflect.DeepEqual(continues, tt.expectedContinue) {
				t.Fatalf(
					"expected continue=%v, got %v",
					tt.expectedContinue,
					continues,
				)
			}

			var gotCont []bool

			for _, in := range tt.inputs {
				gotCont = append(gotCont, engine.Handle(in))
			}

			if len(results) != len(tt.expectedFields) {
				t.Fatalf("len mismatch want=%d got=%d", len(tt.expectedFields), len(results))
			}
			for i := range results {
				if !reflect.DeepEqual(results[i].Fields, tt.expectedFields[i]) {
					t.Fatalf("fields[%d] mismatch\nwant=%v\ngot=%v",
						i, tt.expectedFields[i], results[i].Fields)
				}

				if results[i].IsContext != tt.expectedContext[i] {
					t.Fatalf("ctx[%d] mismatch want=%v got=%v",
						i, tt.expectedContext[i], results[i].IsContext)
				}
			}

			if !reflect.DeepEqual(gotCont, tt.expectedContinue) {
				t.Fatalf("continue mismatch\nwant=%v\ngot=%v", tt.expectedContinue, gotCont)
			}
		})
	}
}
