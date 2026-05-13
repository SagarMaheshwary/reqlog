package scanner

type Config struct {
	Dir         string
	SearchValue string
	IgnoreCase  bool
	Keys        []string
	Since       string
	Limit       int
	Recursive   bool
	Services    []string
	JSONMode    bool
	Latest      bool
	Context     int
}
