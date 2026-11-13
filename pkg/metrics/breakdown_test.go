package metrics

import "testing"

func TestScaleInputTo_KeepsProportionsAndSumsToTotal(t *testing.T) {
	b := TokenBreakdown{SystemTokens: 2000, HistoryTokens: 1500, PromptTokens: 100, ToolTokens: 400}
	b.ScaleInputTo(2000) // exactly half of the 4000 estimated

	want := TokenBreakdown{SystemTokens: 1000, HistoryTokens: 750, PromptTokens: 50, ToolTokens: 200}
	if b != want {
		t.Errorf("scaled = %+v, want %+v", b, want)
	}
}

func TestScaleInputTo_RoundingLeftoverGoesToLargestPart(t *testing.T) {
	b := TokenBreakdown{SystemTokens: 10, HistoryTokens: 5, PromptTokens: 5}
	b.ScaleInputTo(7)

	if sum := b.SystemTokens + b.HistoryTokens + b.PromptTokens + b.ToolTokens; sum != 7 {
		t.Fatalf("parts must sum to the provider's count 7, got %d (%+v)", sum, b)
	}
	if b.SystemTokens != 5 || b.HistoryTokens != 1 || b.PromptTokens != 1 {
		t.Errorf("unexpected split %+v", b)
	}
}

func TestScaleInputTo_LeavesEstimatesWhenNothingToScale(t *testing.T) {
	b := TokenBreakdown{SystemTokens: 30, PromptTokens: 10}
	b.ScaleInputTo(0)
	if b.SystemTokens != 30 || b.PromptTokens != 10 {
		t.Errorf("a zero provider count must leave the estimates alone, got %+v", b)
	}

	empty := TokenBreakdown{}
	empty.ScaleInputTo(500)
	if empty != (TokenBreakdown{}) {
		t.Errorf("nothing estimated means nothing to split, got %+v", empty)
	}
}
