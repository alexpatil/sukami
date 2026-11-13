<div align="center">

<img src="https://repository-images.githubusercontent.com/1432389674/6c14d04e-1580-4125-847c-6d8f59c3b836" alt="Sukami - Open-source AI infrastructure in Go" width="100%" />

<br />
<br />

[Quickstart](#quickstart) • [Usage](#usage) • [Features](#features) • [Configuration](#configuration) • [Benchmarks](#benchmarks) • [Architecture](docs/ARCHITECTURE.md) • [Releases](https://github.com/alexpatil/sukami/releases)

<br />
<br />

**The transparent token optimization data-plane between your applications and upstream LLMs.**
<br />
Sukami is an open-source AI infrastructure proxy in pure Go. It prunes prompt bloat, serves repeated queries from an in-memory cache, and dynamically routes simple queries to economy models.

<br />

[![Release](https://img.shields.io/github/v/release/alexpatil/sukami?style=flat-square&color=blue)](https://github.com/alexpatil/sukami/releases)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io-2496ED?style=flat-square&logo=docker&logoColor=white)](https://github.com/alexpatil/sukami/pkgs/container/sukami)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-6B7280?style=flat-square)](LICENSE)
[![XERJ Corpus Hub contributor](https://hub.xerj.org/contributors/alexpatil/badge.svg)](https://hub.xerj.org/contributors/alexpatil)

</div>

---

- **Zero dependencies**: Built exclusively with the Go standard library, packaged as a single static binary.
- **Drop-in integration**: Works with official OpenAI SDKs by changing only the `base_url`.
- **In-memory cache**: Exact-match response cache. Lexical similarity matching is opt-in.
- **Low overhead**: Pruning, cache lookup, and routing run in-process with no extra network hop.
- **Smart routing**: Opt-in routing of simple prompts to a cheaper model on the same upstream. Complex prompts keep the model you requested.

---

## Quickstart

### Option A: Run with Docker

Run without cloning the repository or installing Go:

```bash
docker run -d -p 8080:8080 -e UPSTREAM_URL=https://api.openai.com/v1 ghcr.io/alexpatil/sukami:latest
```

### Option B: Standalone Binary

Download the pre-compiled binary for Linux, macOS, or Windows from [Releases](https://github.com/alexpatil/sukami/releases):

```bash
# Example for Linux (amd64)
curl -L -o sukami https://github.com/alexpatil/sukami/releases/latest/download/sukami-linux-amd64
chmod +x sukami
./sukami
```

### Option C: Build from Source (Go or Docker Compose)

```bash
git clone https://github.com/alexpatil/sukami.git
cd sukami
# Run with Go
go run ./cmd/sukami

# Or run with Docker Compose
docker compose up -d
```


The gateway starts at `http://localhost:8080`.

---

## Usage

Point your existing OpenAI client to `http://localhost:8080/v1`:

### Python

```python
from openai import OpenAI

client = OpenAI(
    api_key="your-api-key",
    base_url="http://localhost:8080/v1",
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "How do goroutines communicate in Go?"}],
)
print(response.choices[0].message.content)
```

### Node.js / TypeScript

```typescript
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: "your-api-key",
  baseURL: "http://localhost:8080/v1",
});

const response = await client.chat.completions.create({
  model: "gpt-4o",
  messages: [{ role: "user", content: "How do goroutines communicate in Go?" }],
});
console.log(response.choices[0].message.content);
```

### cURL

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "How do goroutines communicate in Go?"}]
  }'
```

Responses are standard OpenAI JSON objects with added observability headers:
- `X-Cache`: `HIT` (served from the cache) or `MISS` (forwarded to upstream).
- `X-Sukami-Pruned-Tokens`: Number of redundant tokens pruned before forwarding.
- `X-Sukami-Retries`: Count of transient upstream retry attempts executed (`0`, `1`, `2`).

---

## Architecture

```text
Client Request (OpenAI SDK / cURL)
        │
        ▼
┌─────────────────────────────┐
│    Auth & Security Guard    │  Validates gateway key and rejects unauthorized access before cache
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│  Smart Router & Optimizer   │  Prompt pruner (preserves code & markdown) + optional complexity router
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│       Response Cache        │  Exact match; lexical similarity is opt-in
└────────────┬────────────────┘
             │ miss
             ▼
┌─────────────────────────────┐
│  Transparent Reverse Proxy  │  Forwards to UPSTREAM_URL or request X-Upstream-Url header
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│   Prometheus Metrics Layer  │  Tokens, latency, and live dynamic cost savings
└─────────────────────────────┘
```

For complete technical specifications, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Features

- **Prompt pruning**: Strips conversational filler and redundant whitespace. Code blocks and markdown indentation are unconditionally preserved.
- **In-memory cache**: Serves a stored response when the same request is sent again. Matching ignores letter case and trailing punctuation.
- **Similarity matching (opt-in)**: With `ENABLE_SIMILARITY_CACHE=true` or the header `X-Sukami-Similarity: true`, a prompt that shares most of its words with a stored prompt is also served from the cache (word-count cosine above 0.85). It is off by default because it compares which words appear, not their order: on the PAWS test set it served the stored answer for 97.54% of pairs labelled as different in meaning. See [docs/BENCHMARKS.md](docs/BENCHMARKS.md).
- **Automatic retries on transient errors**: Transparently retries transient rate limits (`HTTP 429`) and server overloads (`502`, `503`, `504`) with jittered exponential backoff and `Retry-After` header respect. Configurable via `UPSTREAM_MAX_RETRIES` or per-request with `X-Sukami-Max-Retries`.
- **Smart model routing (opt-in)**: Evaluates prompt complexity in-process without external API calls. Routes simple queries to economy models while preserving flagship models for complex code and multi-step reasoning.
- **Cloud security mode**: Set `SUKAMI_API_KEY` to protect the proxy on public networks. Rejects unauthenticated callers before cache access to prevent unauthorized cache inspection.
- **Multi-tenant keys**: Create tenant keys with dedicated rate limits and usage tracking via `/api/keys`.
- **Headless**: No frontend. `GET /` returns JSON service status.
- **Prometheus metrics**: Exports standard Prometheus metrics at `GET /metrics`.

---

## Configuration

Sukami is configured via environment variables, CLI flags, or a local `.env` file:

```bash
cp .env.example .env
```

```ini
# Network & Proxy
PORT=8080                               # Port to listen on (flag: -p, -port)
UPSTREAM_URL=https://api.openai.com/v1  # Upstream provider endpoint (flag: -upstream)

# Cloud Security Mode (optional)
SUKAMI_API_KEY=                         # Master gateway key. Rejects unauthenticated callers before cache

# Dynamic Optimization (can also be toggled per-request via headers)
ENABLE_SMART_ROUTING=false              # Route simple queries to economy models (header: X-Sukami-Smart-Route: true)
ENABLE_OUTPUT_DENSITY=false             # Inject anti-fluff instructions into prompts (header: X-Sukami-Density: true)
ENABLE_SIMILARITY_CACHE=false           # Also serve cached answers for lexically similar prompts (header: X-Sukami-Similarity: true)

# Resilience & Retries (optional)
UPSTREAM_MAX_RETRIES=2                  # Max retries for transient 429/502/503/504 errors (header: X-Sukami-Max-Retries)

# Upstream Fallback (optional: clients typically pass keys via Authorization header)
UPSTREAM_API_KEY=                       # Default upstream API key fallback
```

---

## Benchmarks

`cmd/sukami-bench` sends one fixed prompt directly to your upstream and through the proxy, then prints token counts and latency for each path, including a cache hit and the opt-in output density mode.

```bash
# In .env or the environment
UPSTREAM_URL=https://your-provider.example/v1
UPSTREAM_API_KEY=your-api-key
BENCH_MODEL=the-model-to-test

go run ./cmd/sukami-bench
```

Recorded results and how to read them are in [docs/BENCHMARKS.md](docs/BENCHMARKS.md).

---

## Observability

Standard Prometheus metrics are available at `GET /metrics`:

| Metric Name | Type | Description |
| :--- | :--- | :--- |
| `sukami_requests_total` | Counter | Total proxied requests processed |
| `sukami_cache_hits_total` | Counter | Requests served directly from the cache |
| `sukami_cache_hit_ratio` | Gauge | Real-time cache hit ratio (0.0 to 1.0) |
| `sukami_tokens_saved_total` | Counter | Cumulative input tokens eliminated by pruning |
| `sukami_cost_saved_usd_total` | Counter | Total USD saved computed from live model pricing |

---

## Testing & Verification

```bash
# Run unit tests across all packages
go test -v ./pkg/...

# Run Go static type analysis and vet checks
go vet ./...

# Build standalone production binary
go build -o bin/sukami ./cmd/sukami
```

---

## Community & License

- **Discussions**: [GitHub Discussions](https://github.com/alexpatil/sukami/discussions)
- **Bug Reports**: [GitHub Issues](https://github.com/alexpatil/sukami/issues)
- **Maintainer**: Ganesh Shinde ([codingpenin@gmail.com](mailto:codingpenin@gmail.com))
- **License**: MIT License © Ganesh Shinde. See [LICENSE](LICENSE) for full terms.
