package proxy

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sukami/pkg/auth"
	"sukami/pkg/cache"
	"sukami/pkg/metrics"
	"sukami/pkg/models"
	"sukami/pkg/optimizer"
	"testing"
	"time"
)

const usageWithDetails = `{"prompt_tokens":900,"completion_tokens":120,"total_tokens":1020,"prompt_tokens_details":{"cached_tokens":768},"completion_tokens_details":{"reasoning_tokens":40}}`

func breakdownHandler(t *testing.T, stream bool) (*Handler, *metrics.Tracker) {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if stream {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"id\":\"x\",\"model\":\"m\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"ok\"}}]}\n\n")
			fmt.Fprintf(w, "data: {\"id\":\"x\",\"model\":\"m\",\"choices\":[],\"usage\":%s}\n\n", usageWithDetails)
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"x","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":%s}`, usageWithDetails)
	}))
	t.Cleanup(upstream.Close)

	tracker := metrics.NewTracker(10)
	return NewHandler(cache.NewMemoryCache(time.Hour), tracker, auth.NewMemoryKeyStore(), auth.NewRateLimiter(), upstream.URL), tracker
}

func TestRecord_BreaksDownWhereTokensWent(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprintf("stream=%v", stream), func(t *testing.T) {
			h, tracker := breakdownHandler(t, stream)
			system := "You are a support assistant for an electronics store."
			earlier := "Does the warranty cover water damage?"
			reply := "No. Water damage is excluded under clause four."
			last := "And accidental drops?"

			isoReq{auth: "sk-caller", stream: stream, msgs: []models.ChatMessage{
				{Role: "system", Content: system},
				userMsg(earlier),
				{Role: "assistant", Content: reply},
				userMsg(last),
			}}.send(h)

			got := tracker.GetSummary().TokenBreakdown
			// The estimated parts are scaled to the 900 prompt tokens the upstream billed.
			want := metrics.TokenBreakdown{
				SystemTokens:  int64(optimizer.CountExactTokens("gpt-4o", system)),
				HistoryTokens: int64(optimizer.CountExactTokens("gpt-4o", earlier) + optimizer.CountExactTokens("gpt-4o", reply)),
				PromptTokens:  int64(optimizer.CountExactTokens("gpt-4o", last)),
			}
			want.ScaleInputTo(900)
			want.UpstreamPromptTokens = 900
			want.UpstreamCachedTokens = 768
			want.UpstreamCompletionTokens = 120
			want.UpstreamReasoningTokens = 40
			if got != want {
				t.Errorf("breakdown = %+v\nwant        %+v", got, want)
			}
			if sum := got.SystemTokens + got.HistoryTokens + got.PromptTokens + got.ToolTokens; sum != 900 {
				t.Errorf("input parts must add up to the upstream's 900 prompt tokens, got %d", sum)
			}
			if got.HistoryTokens <= got.PromptTokens {
				t.Errorf("two earlier messages should outweigh the short last prompt: %+v", got)
			}
		})
	}
}

func TestRecord_CacheHitAddsNothingToTheBreakdown(t *testing.T) {
	h, tracker := breakdownHandler(t, false)
	req := isoReq{auth: "sk-caller", msgs: []models.ChatMessage{userMsg("What is the return window?")}}

	req.send(h)
	afterMiss := tracker.GetSummary().TokenBreakdown
	if second := req.send(h); second.Header().Get("X-Cache") != "HIT" {
		t.Fatalf("second identical request should be a cache hit, got X-Cache=%q", second.Header().Get("X-Cache"))
	}

	if got := tracker.GetSummary().TokenBreakdown; got != afterMiss {
		t.Errorf("a cache hit sends nothing upstream, but the breakdown changed: %+v -> %+v", afterMiss, got)
	}
}

func TestPrometheus_ExportsBreakdownByPart(t *testing.T) {
	h, tracker := breakdownHandler(t, false)
	isoReq{auth: "sk-caller", msgs: []models.ChatMessage{userMsg("What is the return window?")}}.send(h)

	out := tracker.FormatPrometheus()
	for _, want := range []string{
		`sukami_tokens_by_part_total{part="upstream_prompt"} 900`,
		`sukami_tokens_by_part_total{part="upstream_cached_prompt"} 768`,
		`sukami_tokens_by_part_total{part="upstream_completion"} 120`,
		`sukami_tokens_by_part_total{part="upstream_reasoning"} 40`,
		`sukami_tokens_by_part_total{part="history"} 0`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("metrics output is missing %q", want)
		}
	}
}
