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
		expectedMessages []string
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
						Message:   "user=123",
						Timestamp: mustParseRFC3339("2024-03-10T12:00:01Z"),
						Service:   "svc",
					},
				},
			},
			expectedMessages: []string{
				"user=123",
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
						Message:   "user=123",
						Timestamp: mustParseRFC3339("2024-03-10T12:00:01Z"),
						Service:   "svc",
					},
				},
				{
					Line:    "2024-03-10T12:00:02Z user=after",
					Service: "svc",
					IsMatch: false,
				},
			},
			expectedMessages: []string{
				"user=before",
				"user=123",
				"user=after",
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
			name:        "stops after trailing context exhausted",
			contextSize: 1,
			cfg: &Config{
				Limit:   1,
				Keys:    []string{"user"},
				Context: 1,
			},
			inputs: []ContextLine{
				{
					Line:    "2024-03-10T12:00:00Z user=123",
					Service: "svc",
					IsMatch: true,
					Entry: &domain.LogEntry{
						Message: "user=123",
					},
				},
				{
					Line:    "2024-03-10T12:00:01Z user=after",
					Service: "svc",
					IsMatch: false,
				},
			},
			expectedMessages: []string{
				"user=123",
				"user=after",
			},
			expectedContext: []bool{
				false,
				true,
			},
			expectedContinue: []bool{
				true,
				false,
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
			expectedMessages: nil,
			expectedContext:  nil,
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

			if len(results) != len(tt.expectedMessages) {
				t.Fatalf(
					"expected %d results, got %d",
					len(tt.expectedMessages),
					len(results),
				)
			}

			for i, result := range results {
				if result.Message != tt.expectedMessages[i] {
					t.Fatalf(
						"result[%d]: expected message %q, got %q. %v",
						i,
						tt.expectedMessages[i],
						result.Message,
						results,
					)
				}

				if result.IsContext != tt.expectedContext[i] {
					t.Fatalf(
						"result[%d]: expected IsContext=%v, got %v",
						i,
						tt.expectedContext[i],
						result.IsContext,
					)
				}
			}
		})
	}
}
