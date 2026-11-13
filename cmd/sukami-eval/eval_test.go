package main

import (
	"strings"
	"sukami/pkg/proxy"
	"testing"
	"time"
)

func TestReadPairs_ParsesLabelledRows(t *testing.T) {
	tsv := "id\tsentence1\tsentence2\tlabel\n" +
		"1\tfirst text\tsecond text\t1\r\n" +
		"\n" +
		"2\tthird text\tfourth text\t0\n"

	pairs, err := readPairs(strings.NewReader(tsv), 0)
	if err != nil {
		t.Fatalf("readPairs: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(pairs))
	}
	if pairs[0] != (pair{first: "first text", second: "second text", same: true}) {
		t.Errorf("unexpected first pair: %+v", pairs[0])
	}
	if pairs[1].same {
		t.Errorf("label 0 must parse as a different-meaning pair")
	}
}

func TestReadPairs_AppliesLimit(t *testing.T) {
	tsv := "id\tsentence1\tsentence2\tlabel\n1\ta\tb\t1\n2\tc\td\t0\n3\te\tf\t1\n"
	pairs, err := readPairs(strings.NewReader(tsv), 2)
	if err != nil {
		t.Fatalf("readPairs: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("expected the limit of 2 pairs, got %d", len(pairs))
	}
}

func TestReadPairs_RejectsMalformedInput(t *testing.T) {
	cases := map[string]string{
		"wrong field count": "id\ts1\ts2\tlabel\n1\tonly two fields\n",
		"bad label":         "id\ts1\ts2\tlabel\n1\ta\tb\tyes\n",
		"header only":       "id\ts1\ts2\tlabel\n",
	}
	for name, tsv := range cases {
		if _, err := readPairs(strings.NewReader(tsv), 0); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// The evaluation runs the real cache, so these pairs are chosen for what the cache does today:
// reordered words are served as a hit, unrelated text is not.
func TestEval_ClassifiesHitsAgainstLabels(t *testing.T) {
	pairs := []pair{
		{first: "The flight from Paris to London was delayed by fog", second: "The flight from London to Paris was delayed by fog", same: false},
		{first: "The committee approved the annual budget on Friday", second: "On Friday the committee approved the annual budget", same: true},
		{first: "Rainfall in the valley peaks during late autumn", second: "Quarterly revenue grew faster than analysts forecast", same: false},
	}

	for name, eval := range map[string]func([]pair, float64) outcome{"isolated": evalIsolated, "shared": evalShared} {
		got := eval(pairs, proxy.SimilarityThreshold)
		if got.samePairs != 1 || got.diffPairs != 2 {
			t.Fatalf("%s: expected 1 same and 2 different pairs, got %d and %d", name, got.samePairs, got.diffPairs)
		}
		if got.trueHits != 1 {
			t.Errorf("%s: the reordered same-meaning pair should be a true hit, got %d", name, got.trueHits)
		}
		if got.falseHits != 1 {
			t.Errorf("%s: the swapped-cities pair should be the only false hit, got %d", name, got.falseHits)
		}
		if got.otherHits != 0 {
			t.Errorf("%s: no lookup should be served another pair's text, got %d", name, got.otherHits)
		}
		if len(got.lookups) != len(pairs) {
			t.Errorf("%s: expected %d timed lookups, got %d", name, len(pairs), len(got.lookups))
		}
	}
}

func TestRenderReport_ShowsCountsAndRates(t *testing.T) {
	results := []result{
		{name: "isolated", outcome: outcome{samePairs: 4, diffPairs: 10, trueHits: 3, falseHits: 5, lookups: []time.Duration{200 * time.Microsecond}}, elapsed: time.Second},
		{name: "shared", outcome: outcome{samePairs: 4, diffPairs: 10, trueHits: 2, falseHits: 4, otherHits: 1, exactHits: 6, lookups: []time.Duration{3 * time.Millisecond}}, elapsed: 2 * time.Second},
	}
	report := renderReport("pairs.tsv", 0.85, results)

	for _, want := range []string{
		"Pairs: 14 from pairs.tsv (4 same meaning, 10 different meaning). Similarity threshold: 0.85.",
		"| isolated | 5 of 10 (50.00%) | 3 of 4 (75.00%) | 0 | 0 | 200µs | 1s |",
		"| shared | 4 of 10 (40.00%) | 2 of 4 (50.00%) | 1 | 6 | 3ms | 2s |",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("report is missing %q\n%s", want, report)
		}
	}
}
