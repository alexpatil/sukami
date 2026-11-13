# Benchmarks

This page reports only what `cmd/sukami-bench` and `cmd/sukami-eval` measured in recorded runs. It makes no projections.

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

## Cache accuracy

`cmd/sukami-eval` runs the real cache over a file of labelled sentence pairs and counts how often a prompt is served the answer stored for a different prompt. It needs no upstream and no API key.

Recorded on 2026-11-13, Windows 11, similarity threshold 0.85 (the value the proxy uses).

Data: PAWS, labelled final test split, 8,000 pairs (3,536 same meaning, 4,464 different meaning). Source: Google, <https://github.com/google-research-datasets/paws>.

| Setup | Different-meaning pairs served the stored answer | Same-meaning pairs served the stored answer | Served another pair's answer (unlabelled) | Same text (exact match) | Median similarity lookup |
| :--- | ---: | ---: | ---: | ---: | ---: |
| One stored prompt per lookup | 4354 of 4464 (97.54%) | 3492 of 3536 (98.76%) | 0 | 14 | 0µs |
| All stored prompts in one cache | 444 of 4464 (9.95%) | 424 of 3536 (11.99%) | 1736 | 5388 | 262.9ms |

What this run shows:

- With one candidate per lookup, the cache served the stored answer for 97.54% of the pairs labelled as different in meaning. It hits same-meaning and different-meaning pairs at almost the same rate, so on this data the similarity match does not tell them apart.
- The cause is the matcher's design. It compares which words appear, not their order, and PAWS pairs are built by swapping words: "the flight from Paris to London" and "the flight from London to Paris" produce the same word counts.
- PAWS is a hard case chosen for that reason. It is not a sample of real prompts, and this run does not say how often real traffic contains such pairs. It does say that when two prompts differ mainly in word order, the cache will usually treat them as the same.
- With every stored prompt in one cache, a similarity lookup that finds no exact match took a median of 262.9ms. The cache held 4,947 distinct prompts; the default capacity is 10,000. Each lookup compares the prompt against every stored entry in its namespace.
- In the second row most lookups (5,388) were exact text matches, because PAWS reuses sentences across pairs; those are correct hits and are not similarity decisions. The 1,736 lookups served another pair's text have no label in the data, so they are reported as a count and not judged. Counts in this row vary slightly between runs when several stored prompts tie on similarity.
- `0µs` means below the resolution of the Windows system clock.

To run it, supply a tab-separated file with a header row and the columns `id`, `sentence1`, `sentence2`, `label` (1 = same meaning):

```bash
go run ./cmd/sukami-eval -pairs path/to/pairs.tsv            # all pairs
go run ./cmd/sukami-eval -pairs path/to/pairs.tsv -limit 500 # a quick pass
```

The second setup compares every lookup against every stored prompt, so its run time grows with the square of the pair count (about five minutes for 8,000 pairs on the machine above). No data set is bundled with the repository.

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
