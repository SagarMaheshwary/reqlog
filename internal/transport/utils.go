package transport

import "strings"

func buildServiceMatcher(services []string) func(string) bool {
	exact := map[string]struct{}{}
	prefixes := []string{}

	for _, s := range services {
		s = strings.TrimSpace(s)

		if s == "" {
			continue
		}

		if before, ok := strings.CutSuffix(s, "*"); ok {
			prefixes = append(prefixes, before)
		} else {
			exact[s] = struct{}{}
		}
	}

	return func(name string) bool {
		if len(exact) == 0 && len(prefixes) == 0 {
			return true
		}

		name = strings.TrimSuffix(name, ".log")

		if _, ok := exact[name]; ok {
			return true
		}

		for _, p := range prefixes {
			if strings.HasPrefix(name, p) {
				return true
			}
		}

		return false
	}
}

func isLogFile(name string) bool {
	return strings.HasSuffix(name, ".log")
}

func quoteShellArg(s string) string {
	if s == "" {
		return "''"
	}

	// Escape single quotes:
	// abc'def -> 'abc'"'"'def'
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
