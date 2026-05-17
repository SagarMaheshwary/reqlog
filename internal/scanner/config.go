package scanner

import "github.com/sagarmaheshwary/reqlog/internal/config"

type Config struct {
	Dir         string
	SearchValue string
	IgnoreCase  bool
	Keys        []string
	Since       string
	Limit       int
	Recursive   bool
	Services    []string
	Latest      bool
	Context     int
	Format      config.LogFormat
}
