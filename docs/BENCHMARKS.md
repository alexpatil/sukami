# Sukami Real-World Performance & Optimization Benchmarks

This document details empirical benchmark results measuring Sukami's token optimization, semantic caching latency, and cost reduction across upstream LLM providers.

---

## 1. Executive Summary

| Execution Path | Output Tokens | Latency | Finish Reason | Cost vs. Direct |
| :--- | :---: | :---: | :---: | :---: |
| **Direct Upstream (Raw)** | 3,072 | 7.33s | `length` (truncated) | 100% (Baseline) |
| **Via Sukami (Optimized)** | 1,979 | 4.50s | `stop` (complete) | **~64% (−35.6%)** |
| **Semantic Cache Hit** | 0 | < 10ms | `stop` (instant) | **0% (−100%)** |

**Net Impact:**
- **38.6% Faster Response Time** (due to reduced token generation volume).
- **35.6% Fewer Billed Tokens** (clean, high-density output without conversational preambles or fluff).
- **Sub-10ms Cache Latency** for semantically identical or near-identical queries.

---

## 2. Four Pillars of Sukami Optimization

Sukami applies optimization across four orthogonal layers:

```text
                  Input Optimization                      Output & Routing
   ┌──────────────────────────────────────────────┐  ┌───────────────────────────────┐
   │ 1. Transparent Prompt Pruning (20–35% input) │  │ 3. Output Density (30–35% out)│
   │ 2. Sub-10ms Semantic Cache (100% savings)    │  │ 4. Smart Model Route (80–95%) │
   └──────────────────────────────────────────────┘  └───────────────────────────────┘
```

### Pillar 1: Transparent Input Prompt Pruning
- Strips conversational boilerplate ("Could you please kindly explain...", "I would like you to...") and collapses redundant whitespace.
- Unconditionally preserves markdown code blocks (`` `...` ``, ```` ```...``` ````) and structural indentation.
- **Observed Input Token Savings:** **20% to 35%**.

### Pillar 2: Sub-10ms Semantic Vector Cache
- Fast-path in-memory cosine similarity matching (>0.85 threshold).
- Protected by **Entity Collision Guard** (distinguishes Docker from Kubernetes) and **Opposites Conflict Guard** (distinguishes `encrypt` from `decrypt`).
- **Observed Hit Latency:** **< 10ms** with 0 upstream API tokens billed.

### Pillar 3: Output Density Optimization
- Injects authoritative anti-fluff directives into system prompts when opted-in (`X-Sukami-Density: true` or `ENABLE_OUTPUT_DENSITY=true`).
- Prevents verbose LLM conversational preambles ("Sure! I would be happy to help...") and sign-offs ("Hope this helps! Feel free to ask...").
- **Observed Output Token Savings:** **30% to 35%**.

### Pillar 4: Prompt Complexity Smart Model Routing
- In-process lexical and syntactic classifier evaluates queries in `<10µs`.
- When enabled (`X-Sukami-Smart-Route: true`), transparently routes simple factual, conversational, and translation queries from expensive flagship models (`gpt-4o`, `claude-3.5-sonnet`) to high-speed economy equivalents (`gpt-4o-mini`, `claude-3.5-haiku`).
- **Observed Cost Savings:** **80% to 95%** on routed queries.

---

## 3. Projected Dollar Savings at Scale

Projected savings calculated against OpenAI `gpt-4o` baseline ($2.50 / 1M input tokens, $10.00 / 1M output tokens):

| Request Volume | Direct Upstream Cost | Via Sukami (Pruning + Density) | Via Sukami (with 25% Cache Hit) | Net Dollar Savings |
| :---: | :---: | :---: | :---: | :---: |
| **10,000 requests** | $32.50 | $21.20 | $15.90 | **$16.60 (51.1%)** |
| **100,000 requests** | $325.00 | $212.00 | $159.00 | **$166.00 (51.1%)** |
| **1,000,000 requests** | $3,250.00 | $2,120.00 | $1,590.00 | **$1,660.00 (51.1%)** |

*(Note: When Smart Model Routing is also enabled for mixed workloads, total savings typically reach 65–85%).*

---

## 4. Reproducing the Benchmark

The benchmark runner is built into the repository using standard Go:

```bash
# Set your API key in .env or environment variable
export GROQ_API_KEY="gsk_..."   # or OPENAI_API_KEY="sk-..."

# Run the benchmark
go run ./cmd/sukami-bench
```

The runner executes back-to-back tests against the upstream provider and prints token counts, latency, and savings percentages directly to the terminal.
