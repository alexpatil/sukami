package main

import (
	"fmt"
	"strings"
	"time"
)

type result struct {
	name    string
	outcome outcome
	elapsed time.Duration
}

func percent(part, whole int) string {
	if whole == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.2f%%", 100*float64(part)/float64(whole))
}

func formatLookup(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	return d.Round(100 * time.Microsecond).String()
}

// renderReport prints the measured counts as Markdown. Every figure is a count from this run.
func renderReport(source string, threshold float64, results []result) string {
	var sb strings.Builder
	first := results[0].outcome
	fmt.Fprintf(&sb, "Pairs: %d from %s (%d same meaning, %d different meaning). Similarity threshold: %.2f.\n\n",
		first.samePairs+first.diffPairs, source, first.samePairs, first.diffPairs, threshold)

	sb.WriteString("| Setup | Different-meaning pairs served the stored answer | Same-meaning pairs served the stored answer | Served another pair's answer (unlabelled) | Same text (exact match) | Median similarity lookup | Run time |\n")
	sb.WriteString("| :--- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, r := range results {
		o := r.outcome
		fmt.Fprintf(&sb, "| %s | %d of %d (%s) | %d of %d (%s) | %d | %d | %s | %s |\n",
			r.name,
			o.falseHits, o.diffPairs, percent(o.falseHits, o.diffPairs),
			o.trueHits, o.samePairs, percent(o.trueHits, o.samePairs),
			o.otherHits, o.exactHits, formatLookup(o.medianLookup()), r.elapsed.Round(time.Millisecond))
	}
	return sb.String()
}
