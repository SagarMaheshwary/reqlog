package scanner

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/sagarmaheshwary/reqlog/internal/domain"
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
