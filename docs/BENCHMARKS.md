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

## Multi-turn conversation

`go run ./cmd/sukami-bench -scenario chat` plays one five-turn support conversation on three paths. Every turn resends the system prompt and the whole history, as a chat client does, and the upstream's real answers become the history of the next turn.

Recorded on Windows 11 with `BENCH_PAUSE=45s`, from a local build that combines this runner with the changes in #63, #65 and #67.

Date: 2026-11-13. Model: `openai/gpt-oss-120b`. Upstream: `api.groq.com`. Conversation: 5 turns. Runs per path: 2.
Each cell is the total over all turns of one conversation, as billed by the upstream: the median, with the range across runs in parentheses.

| Path | Prompt tokens | of which provider-cached | Completion tokens | of which reasoning | Prompt + completion | Finish reasons |
| :--- | :---: | :---: | :---: | :---: | :---: | :---: |
| Direct to upstream | 11573 (10638 to 12508) | 768 (0 to 1536) | 3272 (2962 to 3583) | 493 (435 to 552) | 14845 (13600 to 16091) | `stop` |
| Via Sukami (defaults) | 11445 (10412 to 12478) | 512 (0 to 1024) | 3105 (2647 to 3563) | 493 (437 to 550) | 14550 (13059 to 16041) | `stop` |
| Via Sukami with X-Sukami-Density: true | 9469 (9309 to 9629) | 0 | 1929 (1782 to 2076) | 445 (369 to 521) | 11398 (11091 to 11705) | `stop` |

Where the billed prompt tokens went: the upstream's prompt count for each request, split in the proportions the proxy estimates.

| Path | System prompt | History (earlier turns) | New prompt | Tools |
| :--- | :---: | :---: | :---: | :---: |
| Via Sukami (defaults) | 6242 (6179 to 6306) | 5113 (4016 to 6211) | 89 (88 to 90) | 0 |
| Via Sukami with X-Sukami-Density: true | 6564 (6564 to 6565) | 2814 (2654 to 2975) | 90 | 0 |

What this run shows:

- With default settings the proxy did not lower the bill. The proxied total (14550) and the direct total (14845) differ by less than the spread between two runs of the same path (13600 to 16091 for direct).
- With output density on, the conversation was billed 11398 tokens against 14845 direct, 23% fewer at the medians. Completion tokens fell from 3272 to 1929. Prompt tokens fell from 11573 to 9469, although the density directive adds tokens to the system prompt, because shorter answers make a shorter history to resend on every later turn. Every answer finished with `stop` on all paths.
- This run does not measure answer quality. Shorter answers may leave out detail a user wanted.
- Of the prompt tokens billed for the conversation, about 55% were the system prompt resent five times and about 45% were earlier turns. The user's new messages were under 1%. Pruning filler from the user's text therefore works on the smallest part of the bill.
- The provider's own prompt cache hit on some requests and not on others, on the direct path and through the proxy alike (0 to 1536 cached tokens per conversation). Two runs are too few to compare the paths on this. Groq documents cache hits as not guaranteed.
- `openai/gpt-oss-120b` is a reasoning model, so completion counts include reasoning tokens.

The bundled conversation is `cmd/sukami-bench/chat_scenario.json`. Set `BENCH_CHAT_FILE` to measure your own: a JSON object with a `system` string and a `turns` array of user messages. One run makes fifteen billed upstream calls with the bundled conversation.

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
