# Benchmarks

This page reports only what `cmd/sukami-bench` measured in a recorded run. It makes no projections.

## Results

Recorded with the runner at commit `a07df82`, on Windows 11, with `BENCH_PAUSE=40s`.

Date: 2026-11-13. Model: `openai/gpt-oss-120b`. Upstream: `api.groq.com`. Runs per case: 3.
Cells show the median, with the range across runs in parentheses.

| Case | Prompt tokens | Completion tokens | Latency | Finish reasons |
| :--- | :---: | :---: | :---: | :---: |
| Direct to upstream | 181 | 3072 | 7.00s (6.99s to 7.22s) | `length` |
| Via Sukami (defaults) | 181 | 3072 | 7.39s (7.08s to 8.20s) | `length` |
| Via Sukami, same request again (cache hit) | 181 | 3072 | 0µs (0µs to 737µs) | `length` |
| Via Sukami with X-Sukami-Density: true | 246 | 1864 (1575 to 2277) | 4.65s (3.94s to 5.85s) | `stop` |

What this run shows:

- With default settings the proxy changed nothing about this request. The upstream billed the same 181 prompt tokens, because the prompt has no filler for the pruner to remove. The 0.39s difference in median latency is inside the spread of the proxied runs (7.08s to 8.20s), so three runs do not separate proxy overhead from upstream variance.
- A cache hit returned in under one millisecond without calling the upstream. The `0µs` cells mean the request finished below the resolution of the Windows system clock, not in zero time.
- The direct answer was cut off at the upstream's 3072-token output limit in every run. With output density on, the answer finished on its own at a median of 1864 completion tokens, for 65 extra prompt tokens. Because the baseline is truncated, this run cannot say how many tokens density saves against a complete baseline answer.
- `openai/gpt-oss-120b` is a reasoning model, so its completion token counts include reasoning tokens.

Earlier versions of this page listed token and latency figures, percentage ranges, and dollar projections. They were removed because the old runner never sent a request through the proxy, and no record of the runs behind them exists in the repository.

## What the runner measures

The runner starts the real chat completions handler in-process on a loopback port and sends one fixed prompt (a PostgreSQL vs MongoDB comparison) through four cases:

| Case | What it shows |
| :--- | :--- |
| Direct to upstream | The baseline, with no proxy involved |
| Via Sukami (defaults) | Prompt pruning plus the proxy's own overhead, on a cache miss |
| Via Sukami, same request again | Latency of a cache hit. The upstream is not called |
| Via Sukami with `X-Sukami-Density: true` | The opt-in output density directive |

Every proxied case starts from an empty cache. Token counts are the upstream's own `usage` numbers, not estimates. Latency is wall-clock time for the whole non-streaming request, measured on the client side.

Each case runs `BENCH_RUNS` times. The report shows the median, with the range across runs in parentheses.

## How to read the numbers

- Model output varies from call to call. Compare medians, and treat a difference smaller than the range as noise.
- If a case reports the finish reason `length`, the upstream cut the answer off at its output limit. A truncated baseline makes any saving against it look smaller than it is.
- The fixed prompt contains little conversational filler, so it says little about prompt pruning. Pruning savings depend on how much filler your own prompts carry.
- Smart model routing is not measured. Its effect is the price difference between the requested model and the one it routes to, which depends on your provider.

## Running it

```bash
# In .env or the environment
UPSTREAM_URL=https://your-provider.example/v1
UPSTREAM_API_KEY=your-api-key
BENCH_MODEL=the-model-to-test
BENCH_RUNS=3            # optional, default 3
BENCH_PAUSE=30s         # optional wait after each upstream call, for rate-limited keys

go run ./cmd/sukami-bench
```

One run makes three billed upstream calls (the cache hit is free), so the default makes nine. The report is printed as a Markdown table on stdout.

The benchmark proxy runs with upstream retries turned off, so a rate-limit response fails the run instead of being hidden inside a latency figure. If that happens, set `BENCH_PAUSE`. The pause is not part of any measured latency.
