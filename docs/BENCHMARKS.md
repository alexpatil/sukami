# Benchmarks

This page reports only what `cmd/sukami-bench` measured in a recorded run. It makes no projections.

## Results

No run has been recorded with the current runner yet. Run it as described below and paste its output here, together with the commit it ran on.

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

go run ./cmd/sukami-bench
```

One run makes three billed upstream calls (the cache hit is free), so the default makes nine. The report is printed as a Markdown table on stdout.
