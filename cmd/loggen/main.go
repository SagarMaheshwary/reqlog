package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

func main() {
	var (
		out        string
		lines      int
		jsonMode   bool
		matchEvery int
	)

	flag.StringVar(&out, "out", "test.log", "output file")
	flag.IntVar(&lines, "lines", 100000, "number of lines")
	flag.BoolVar(&jsonMode, "json", false, "generate json logs")
	flag.IntVar(&matchEvery, "match-every", 1000, "insert matching request every N lines")
	flag.Parse()

	f, err := os.Create(out)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	baseTime := time.Now().UTC()
	targetID := "bench-target"

	for i := 0; i < lines; i++ {
		ts := baseTime.Add(time.Duration(i) * time.Millisecond).Format(time.RFC3339)
		reqID := "other-id"
		if matchEvery > 0 && i%matchEvery == 0 {
			reqID = targetID
		}

		if jsonMode {
			fmt.Fprintf(f, `{"time":"%s","request_id":"%s","message":"message-%d"}`+"\n", ts, reqID, i)
		} else {
			fmt.Fprintf(f, "%s request_id=%s message-%d\n", ts, reqID, i)
		}
	}
}
