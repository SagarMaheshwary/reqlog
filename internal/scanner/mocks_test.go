package scanner

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
	"github.com/sagarmaheshwary/reqlog/internal/transport"
)

type mockDockerClient struct {
	logsFn func(container string, follow bool, since string) (io.ReadCloser, error)
	listFn func() ([]string, error)
}

func (m *mockDockerClient) Logs(container string, follow bool, since string) (io.ReadCloser, error) {
	return m.logsFn(container, follow, since)
}

func (m *mockDockerClient) ListContainers() ([]string, error) {
	return m.listFn()
}

type testFormatter struct{}

func (f *testFormatter) Format(entry domain.LogEntry) string {
	keys := make([]string, 0, len(entry.Fields))

	for k := range entry.Fields {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var fields []string

	for _, k := range keys {
		fields = append(fields, fmt.Sprintf("%s=%v", k, entry.Fields[k]))
	}

	return fmt.Sprintf(
		"%s [%s] %s %s",
		entry.Timestamp.Format(time.RFC3339),
		entry.Service,
		entry.Message,
		strings.Join(fields, " "),
	)
}

type mockLogFileReader struct {
	openFn           func(context.Context, string) (io.ReadCloser, error)
	openFromOffsetFn func(context.Context, string, int64) (io.ReadCloser, error)
	sizeFn           func(context.Context, string) (int64, error)
	listFilesFn      func(context.Context, string, transport.ListOptions) ([]string, error)
}

func (m *mockLogFileReader) Open(ctx context.Context, path string) (io.ReadCloser, error) {
	return m.openFn(ctx, path)
}

func (m *mockLogFileReader) OpenFromOffset(ctx context.Context, path string, offset int64) (io.ReadCloser, error) {
	return m.openFromOffsetFn(ctx, path, offset)
}

func (m *mockLogFileReader) Size(ctx context.Context, path string) (int64, error) {
	return m.sizeFn(ctx, path)
}

func (m *mockLogFileReader) ListFiles(
	ctx context.Context,
	dir string,
	opts transport.ListOptions,
) ([]string, error) {
	return m.listFilesFn(ctx, dir, opts)
}
