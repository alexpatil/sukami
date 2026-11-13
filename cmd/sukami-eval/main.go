// Command sukami-eval measures how often the response cache serves an answer for a prompt
// that means something different, using a file of labelled sentence pairs. It needs no upstream.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sukami/pkg/proxy"
	"time"
)

func main() {
	pairsPath := flag.String("pairs", "", "TSV file with a header row and columns id, sentence1, sentence2, label (1 = same meaning)")
	limit := flag.Int("limit", 0, "use only the first N pairs (0 = all)")
	threshold := flag.Float64("threshold", proxy.SimilarityThreshold, "similarity threshold; defaults to the value the proxy uses")
	flag.Parse()

	if *pairsPath == "" {
		fmt.Fprintln(os.Stderr, "usage: sukami-eval -pairs <file.tsv> [-limit N] [-threshold T]")
		os.Exit(2)
	}
	file, err := os.Open(*pairsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	defer file.Close()

	pairs, err := readPairs(file, *limit)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	results := []result{
		timed("One stored prompt per lookup", func() outcome { return evalIsolated(pairs, *threshold) }),
		timed("All stored prompts in one cache", func() outcome { return evalShared(pairs, *threshold) }),
	}
	fmt.Print(renderReport(filepath.Base(*pairsPath), *threshold, results))
}

func timed(name string, run func() outcome) result {
	start := time.Now()
	o := run()
	return result{name: name, outcome: o, elapsed: time.Since(start)}
}
