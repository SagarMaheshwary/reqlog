package scanner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/config"
	"github.com/sagarmaheshwary/reqlog/internal/docker"
	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/formatter"
)

var errTest = errors.New("docker error")

func dockerLogs(lines []string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(strings.Join(lines, "\n")))
}

func newTestDockerScanner(cfg *Config, client docker.DockerClient) *DockerScanner {
	lp := NewLineProcessor(cfg, NewTimeParser())
	return NewDockerScanner(lp, client, io.Discard, io.Discard, time.Now())
}

func TestDockerScanner_Scan(t *testing.T) {
	now := time.Now().UTC()
	oldTime := now.Add(-10 * time.Minute).Format(time.RFC3339)
	newTime := now.Add(-1 * time.Minute).Format(time.RFC3339)

	tests := []struct {
		name       string
		logsFn     func(container string, follow bool, since string) (io.ReadCloser, error)
		cfg        *Config
		containers []string
		want       int
		wantErrLog string
	}{
		{
			name: "single container logs",
			logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return dockerLogs([]string{
					"2024-03-10T12:00:00Z user=123 status=ok",
					"2024-03-10T12:01:00Z user=456 status=fail",
					"2024-03-10T12:02:00Z user=123 status=ok",
				}), nil
			},
			cfg:        &Config{SearchValue: "123", Keys: []string{"user"}},
			containers: []string{"auth"},
			want:       2,
		},
		{
			name: "with since filter",
			logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return dockerLogs([]string{
					oldTime + " user=123",
					newTime + " user=123",
				}), nil
			},
			cfg:        &Config{SearchValue: "123", Keys: []string{"user"}, Since: "5m"},
			containers: []string{"svc"},
			want:       1,
		},
		{
			name: "ignore case",
			logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return dockerLogs([]string{"2024-03-10T12:00:00Z user=ABC"}), nil
			},
			cfg:        &Config{SearchValue: "abc", Keys: []string{"user"}, IgnoreCase: true},
			containers: []string{"svc"},
			want:       1,
		},
		{
			name: "multi container with limit",
			logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return dockerLogs([]string{"2024-03-10T12:00:00Z id=123", "2024-03-10T12:00:00Z id=123", "2024-03-10T12:00:00Z id=123"}), nil
			},
			cfg:        &Config{SearchValue: "123", Keys: []string{"id"}, Limit: 2},
			containers: []string{"a", "b"},
			want:       2,
		},
		{
			name: "skips container errors",
			logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
				if container == "bad" {
					return nil, fmt.Errorf("docker error")
				}
				return dockerLogs([]string{"2024-03-10T12:00:00Z id=123"}), nil
			},
			cfg:        &Config{SearchValue: "123", Keys: []string{"id"}},
			containers: []string{"good", "bad"},
			want:       1,
		},
		{
			name: "logs error output",
			logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return nil, fmt.Errorf("docker scan error")
			},
			cfg:        &Config{SearchValue: "123", Keys: []string{"user"}},
			containers: []string{"auth"},
			want:       0,
			wantErrLog: "docker scan error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDockerClient{
				logsFn: tt.logsFn,
				listFn: func() ([]string, error) { return tt.containers, nil },
			}

			lp := NewLineProcessor(tt.cfg, NewTimeParser())
			var out, errOut bytes.Buffer
			ds := NewDockerScanner(lp, mock, &out, &errOut, time.Now())

			results, err := ds.Scan(tt.containers)
			if err != nil {
				t.Fatal(err)
			}

			if len(results) != tt.want {
				fmt.Println(results)
				t.Errorf("expected %d results, got %d", tt.want, len(results))
			}

			if tt.wantErrLog != "" && !strings.Contains(errOut.String(), tt.wantErrLog) {
				t.Errorf("expected error log containing %q, got %q", tt.wantErrLog, errOut.String())
			}
		})
	}
}

func TestDockerScanner_Scan_InvalidSince(t *testing.T) {
	cfg := &Config{
		SearchValue: "abc",
		Keys:        []string{"user"},
		IgnoreCase:  true,
		Since:       "invalid",
	}

	ds := newTestDockerScanner(cfg, &mockDockerClient{})

	_, err := ds.Scan([]string{"auth"})
	if err == nil {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestDockerScanner_Scan_JSON(t *testing.T) {
	tests := []struct {
		name        string
		container   string
		logLines    []string
		expectedLen int
		assert      func(t *testing.T, results []domain.LogEntry)
	}{
		{
			name:      "valid json logs",
			container: "auth",
			logLines: []string{
				`{"time":"2024-03-10T12:00:00Z","user":"123","status":"ok"}`,
				`{"time":"2024-03-10T12:00:00Z","user":"456","status":"fail"}`,
				`{"time":"2024-03-10T12:00:00Z","user":"123","status":"ok"}`,
			},
			expectedLen: 2,
			assert: func(t *testing.T, results []domain.LogEntry) {
				for _, r := range results {
					if r.Service != "auth" {
						t.Errorf("expected service auth, got %s", r.Service)
					}
				}
			},
		},
		{
			name:      "invalid json lines are skipped",
			container: "svc",
			logLines: []string{
				`{"time":"2024-03-10T12:00:00Z","user":"123"}`,
				`invalid json`,
				`{"time":"2024-03-10T12:00:00Z","user":"123"}`,
			},
			expectedLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDockerClient{
				logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
					return dockerLogs(tt.logLines), nil
				},
			}

			cfg := &Config{
				SearchValue: "123",
				Keys:        []string{"user"},
				Format:      config.FormatJSON,
			}

			ds := newTestDockerScanner(cfg, mock)

			results, err := ds.Scan([]string{tt.container})
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != tt.expectedLen {
				t.Fatalf("expected %d results, got %d", tt.expectedLen, len(results))
			}

			if tt.assert != nil {
				tt.assert(t, results)
			}
		})
	}
}

func TestDockerScanner_Scan_Latest(t *testing.T) {
	mock := &mockDockerClient{
		logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
			switch container {
			case "a":
				return dockerLogs([]string{
					"2024-03-10T12:00:01Z id=123",
					"2024-03-10T12:00:03Z id=123",
					"2024-03-10T12:00:05Z id=123",
				}), nil

			case "b":
				return dockerLogs([]string{
					"2024-03-10T12:00:02Z id=123",
					"2024-03-10T12:00:04Z id=123",
					"2024-03-10T12:00:06Z id=123",
				}), nil
			}

			return dockerLogs(nil), nil
		},
		listFn: func() ([]string, error) {
			return []string{"a", "b"}, nil
		},
	}

	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"id"},
		Limit:       2,
		Latest:      true,
	}

	lp := NewLineProcessor(cfg, NewTimeParser())

	var out, errOut bytes.Buffer
	ds := NewDockerScanner(lp, mock, &out, &errOut, time.Now())

	results, err := ds.Scan([]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	expected := []string{
		"2024-03-10T12:00:05Z",
		"2024-03-10T12:00:06Z",
	}

	for i, ts := range expected {
		expectedTime, err := time.Parse(time.RFC3339, ts)
		if err != nil {
			t.Fatal(err)
		}

		if !results[i].Timestamp.Equal(expectedTime) {
			t.Fatalf(
				"result[%d]: expected %v, got %v",
				i,
				expectedTime,
				results[i].Timestamp,
			)
		}
	}
}

func TestDockerScanner_Scan_Context(t *testing.T) {
	mock := &mockDockerClient{
		logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
			return dockerLogs([]string{
				"2024-03-10T12:00:00Z user=ignore",
				"2024-03-10T12:00:01Z user=123",
				"2024-03-10T12:00:02Z user=ignore",
				"2024-03-10T12:00:03Z user=ignore",
				"2024-03-10T12:00:04Z user=123", // should not be processed
			}), nil
		},
		listFn: func() ([]string, error) {
			return []string{"auth"}, nil
		},
	}

	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"user"},
		Context:     2,
		Limit:       1,
	}

	lp := NewLineProcessor(cfg, NewTimeParser())

	var out, errOut bytes.Buffer

	ds := NewDockerScanner(
		lp,
		mock,
		&out,
		&errOut,
		time.Now(),
	)

	results, err := ds.Scan([]string{"auth"})
	if err != nil {
		t.Fatal(err)
	}

	if len(results) != 4 {
		t.Fatalf("expected 4 results, got %d", len(results))
	}

	expected := []struct {
		raw       string
		isContext bool
	}{
		{
			raw:       "2024-03-10T12:00:00Z user=ignore",
			isContext: true,
		},
		{
			raw:       "2024-03-10T12:00:01Z user=123",
			isContext: false,
		},
		{
			raw:       "2024-03-10T12:00:02Z user=ignore",
			isContext: true,
		},
		{
			raw:       "2024-03-10T12:00:03Z user=ignore",
			isContext: true,
		},
	}

	for i, exp := range expected {
		if results[i].Raw != exp.raw {
			t.Fatalf(
				"result[%d]: expected raw %q, got %q",
				i,
				exp.raw,
				results[i].Raw,
			)
		}

		if results[i].IsContext != exp.isContext {
			t.Fatalf(
				"result[%d]: expected IsContext=%v, got %v",
				i,
				exp.isContext,
				results[i].IsContext,
			)
		}
	}
}

func TestDockerScanner_ListSources(t *testing.T) {
	tests := []struct {
		name       string
		containers []string
		services   []string
		want       []string
	}{
		{
			name:       "no filter",
			containers: []string{"auth", "db"},
			services:   []string{},
			want:       []string{"auth", "db"},
		},
		{
			name:       "exact match",
			containers: []string{"auth", "db"},
			services:   []string{"auth", " "}, //skip empty strings
			want:       []string{"auth"},
		},
		{
			name:       "wildcard",
			containers: []string{"svc", "svc-1", "db"},
			services:   []string{"svc*"},
			want:       []string{"svc", "svc-1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDockerClient{
				listFn: func() ([]string, error) {
					return tt.containers, nil
				},
			}

			cfg := &Config{
				Services: tt.services,
			}

			ds := newTestDockerScanner(cfg, mock)

			got, err := ds.ListSources()
			if err != nil {
				t.Fatal(err)
			}

			sort.Strings(got)
			sort.Strings(tt.want)

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestDockerScanner_ListSources_Error(t *testing.T) {
	mock := &mockDockerClient{
		listFn: func() ([]string, error) {
			return nil, errors.New("list error")
		},
	}
	ds := newTestDockerScanner(&Config{}, mock)

	_, err := ds.ListSources()
	if err == nil {
		t.Fatalf("expected error, got %v", err)
	}
}

func TestDockerScanner_Follow(t *testing.T) {
	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"user"},
	}
	lp := NewLineProcessor(cfg, NewTimeParser())

	tests := []struct {
		name       string
		clientLogs func(container string, follow bool, since string) (io.ReadCloser, error)
		clientList func() ([]string, error)
		containers []string
		want       []string
	}{
		{
			name: "single container logs",
			clientLogs: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(
					"2024-03-10T12:00:00Z user=123 status=ok",
				)), nil
			},
			clientList: func() ([]string, error) { return []string{"auth"}, nil },
			containers: []string{"auth"},
			want: []string{
				"2024-03-10T12:00:00Z [auth]",
			},
		},
		{
			name: "multiple containers",
			clientLogs: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(
					"2024-03-10T12:00:00Z user=123",
				)), nil
			},
			clientList: func() ([]string, error) { return []string{"auth", "db"}, nil },
			containers: []string{"auth", "db"},
			want: []string{
				"[auth]",
				"[db]",
			},
		},
		{
			name: "ignore non-matching lines",
			clientLogs: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(
					"2024-03-10T12:00:00Z user=123\n" +
						"2024-03-10T12:00:00Z other=xyz",
				)), nil
			},
			clientList: func() ([]string, error) { return []string{"svc"}, nil },
			containers: []string{"svc"},
			want: []string{
				"[svc]",
			},
		},
		{
			name: "container returns error",
			clientLogs: func(container string, follow bool, since string) (io.ReadCloser, error) {
				return nil, errTest
			},
			clientList: func() ([]string, error) { return []string{"auth"}, nil },
			containers: []string{"auth"},
			want:       []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mockDockerClient{
				logsFn: tt.clientLogs,
				listFn: tt.clientList,
			}

			var out bytes.Buffer
			ds := NewDockerScanner(lp, client, &out, io.Discard, time.Now())

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			ds.Follow(ctx, tt.containers, &testFormatter{})

			lines := strings.FieldsFunc(out.String(), func(r rune) bool {
				return r == '\n' || r == '\r'
			})

			sort.Strings(lines)
			want := append([]string(nil), tt.want...)
			sort.Strings(want)

			if len(lines) != len(want) {
				t.Fatalf("expected %d lines, got %d", len(want), len(lines))
			}

			for i := range lines {
				if !strings.Contains(lines[i], want[i]) {
					t.Errorf("line %d: expected to contain %q, got %q", i, want[i], lines[i])
				}
			}
		})
	}
}

func TestDockerScanner_Follow_Errors(t *testing.T) {
	cfg := &Config{
		SearchValue: "123",
		Keys:        []string{"user"},
	}
	lp := NewLineProcessor(cfg, NewTimeParser())

	client := &mockDockerClient{
		logsFn: func(container string, follow bool, since string) (io.ReadCloser, error) {
			return nil, fmt.Errorf("docker error for %s", container)
		},
		listFn: func() ([]string, error) { return []string{"auth"}, nil },
	}

	var out bytes.Buffer
	var errOut bytes.Buffer

	ds := NewDockerScanner(lp, client, &out, &errOut, time.Now())

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	f := formatter.NewFormatter(&formatter.Opts{
		Output: config.OutputPretty,
	})
	ds.Follow(ctx, []string{"auth"}, f)

	if !strings.Contains(errOut.String(), "docker error for auth") {
		t.Errorf("expected error log, got %q", errOut.String())
	}
}
