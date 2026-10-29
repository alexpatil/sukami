<div align="center">

# ⚡ Sukami

**OpenAI-compatible caching proxy and token optimizer in pure Go.**

[![CI](https://img.shields.io/github/actions/workflow/status/alexpatil/sukami/ci.yml?branch=main&style=flat-square&logo=githubactions&logoColor=white&label=CI)](https://github.com/alexpatil/sukami/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/alexpatil/sukami?style=flat-square&logo=github&color=blue)](https://github.com/alexpatil/sukami/releases)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-6B7280?style=flat-square)](LICENSE)
[![Zero Dependencies](https://img.shields.io/badge/dependencies-zero-4ADE80?style=flat-square)](go.mod)
[![Docker](https://img.shields.io/badge/Docker-~10MB-0EA5E9?style=flat-square&logo=docker&logoColor=white)](https://github.com/alexpatil/sukami/pkgs/container/sukami)

<br />

[Quickstart](#quickstart) • [Usage](#usage) • [Features](#features) • [Configuration](#configuration) • [Benchmarks](#benchmarks) • [Architecture](docs/ARCHITECTURE.md)

</div>

---

Sukami sits between your application and upstream LLM providers (OpenAI, Groq, DeepSeek, OpenRouter, Ollama). It prunes conversational filler from prompts, serves repeated queries from an in-memory vector cache in under 10ms, and exports Prometheus metrics—without changing your application logic.

- **Zero dependencies** — Built exclusively with the Go standard library, single static binary (~10MB).
- **Drop-in integration** — Works with official OpenAI SDKs by changing only the `base_url`.
- **In-memory cache** — Cosine vector similarity with entity and opposites collision guards.
- **Fast execution** — Pruning runs in <50µs; routing decisions in <10µs; cache hits in <10ms.

---

## Quickstart

### Option A: Standalone Binary

Download the pre-compiled binary for Linux, macOS, or Windows from [Releases](https://github.com/alexpatil/sukami/releases):

```bash
# Example for Linux (amd64)
curl -L -o sukami https://github.com/alexpatil/sukami/releases/latest/download/sukami-linux-amd64
chmod +x sukami
./sukami
```

### Option B: Run with Go (1.22+)

```bash
git clone https://github.com/alexpatil/sukami.git
cd sukami
cp .env.example .env  # optional configuration
go run ./cmd/sukami
```

### Option C: Run with Docker

```bash
docker compose up -d
```

The gateway and developer dashboard start at `http://localhost:8080`.

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

The response is a standard OpenAI JSON object with zero wrapper fields.

---

## Architecture

```text
Client Request (OpenAI SDK / cURL)
        │
        ▼
┌─────────────────────────────┐
│    Auth & Security Guard    │  Cloud Security Mode (SUKAMI_API_KEY) & Zero Cache Leak
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│  Smart Router & Optimizer   │  <10µs classifier + filler pruner (preserves code & markdown)
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│    Semantic Vector Cache    │  Cosine similarity (>0.85) + collision guards
└────────────┬────────────────┘
             │ miss
             ▼
┌─────────────────────────────┐
│    Multi-Provider Router    │  OpenAI · Groq · DeepSeek · OpenRouter · Ollama
└────────────┬────────────────┘
             │
             ▼
┌─────────────────────────────┐
│   Prometheus Metrics Layer  │  Tokens · latency · live dynamic USD savings
└─────────────────────────────┘
```

For complete technical specifications, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## Features

- **Prompt Pruning** — Strips conversational filler ("Please kindly explain...") and redundant whitespace. Code blocks and markdown indentation are unconditionally preserved.
- **In-Memory Semantic Cache** — Caches responses with cosine similarity matching (>0.85). Built-in entity and opposites guards prevent false positives between distinct concepts (e.g. `Docker` vs `Kubernetes`, `encrypt` vs `decrypt`).
- **Smart Model Routing (Opt-in)** — Evaluates prompt complexity in <10µs without external API calls. Transparently routes simple queries from `gpt-4o` to `gpt-4o-mini` while preserving flagship models for complex code and multi-step reasoning.
- **Cloud Security Mode** — Set `SUKAMI_API_KEY` to lock down the proxy on public servers. Rejects unauthenticated callers before cache access (Zero Cache Leak guarantee).
- **Multi-Tenant Keys** — Create tenant keys with dedicated rate limits and usage tracking via `/api/keys`.
- **Developer Dashboard** — Embedded single-page dashboard and live testing sandbox accessible at `http://localhost:8080`.
- **Prometheus Metrics** — Exports standard Prometheus 0.0.4 metrics at `GET /metrics`.

---

## Configuration

Sukami is configured via environment variables, CLI flags, or a local `.env` file. Settings can also be toggled dynamically per-request using HTTP headers.

### Gateway & Optimization

| Variable | CLI Flag | Request Header | Default | Description |
| :--- | :--- | :--- | :--- | :--- |
| `PORT` | `-port`, `-p` | — | `8080` | Port to listen on |
| `UPSTREAM_URL` | `-upstream` | — | `https://api.openai.com/v1` | Default fallback completions endpoint |
| `SUKAMI_API_KEY` | — | `Authorization: Bearer <key>`<br>`X-Sukami-Master-Key` | *(none)* | Master gateway key. Enables Cloud Security Mode |
| `ENABLE_SMART_ROUTING` | — | `X-Sukami-Smart-Route: true` | `false` | Enables <10µs model complexity routing |
| `ENABLE_OUTPUT_DENSITY` | — | `X-Sukami-Density: true` | `false` | Injects concise anti-fluff system prompt directive |

### Upstream Provider Fallbacks

Clients typically supply upstream provider credentials directly in the `Authorization: Bearer <key>` request header. If omitted, Sukami falls back to server-level environment keys:

| Variable | Supported Models | Provider Endpoint |
| :--- | :--- | :--- |
| `OPENAI_API_KEY` | `gpt-4o`, `gpt-4o-mini`, `o1`, `o3-mini`, `text-embedding-3-*` | `https://api.openai.com/v1` |
| `GROQ_API_KEY` | `llama-3.3-70b-versatile`, `llama-3.1-8b-instant`, `mixtral-8x7b-32768` | `https://api.groq.com/openai/v1` |
| `DEEPSEEK_API_KEY` | `deepseek-chat`, `deepseek-reasoner` | `https://api.deepseek.com/v1` |
| `OPENROUTER_API_KEY` | `anthropic/*`, `google/*`, `meta-llama/*` | `https://openrouter.ai/api/v1` |

Copy `.env.example` to get started:

```bash
cp .env.example .env
```

---

## Benchmarks

Workload: technical comparison (PostgreSQL vs. MongoDB).

| Execution Path | Output Tokens | Latency | Finish Reason | Cost vs. Direct |
| :--- | :---: | :---: | :---: | :---: |
| **Direct Upstream (Raw)** | 3,072 | 7.33s | `length` (truncated) | 100% (Baseline) |
| **Via Sukami (Pruned)** | 1,979 | 4.50s | `stop` (complete) | **~64% (−35.6%)** |
| **Semantic Cache Hit** | 0 | < 10ms | `stop` (instant) | **0% (−100%)** |

To reproduce against upstream providers:

```bash
export OPENAI_API_KEY="sk-..."
go run ./cmd/sukami-bench
```

For detailed empirical methodology, see [docs/BENCHMARKS.md](docs/BENCHMARKS.md).

---

## Observability

Standard Prometheus metrics are available at `GET /metrics`:

| Metric Name | Type | Description |
| :--- | :--- | :--- |
| `sukami_requests_total` | Counter | Total proxied requests processed |
| `sukami_cache_hits_total` | Counter | Requests served directly from the semantic cache |
| `sukami_cache_hit_ratio` | Gauge | Real-time semantic cache hit ratio (0.0 to 1.0) |
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

## Roadmap

- [ ] **Multi-Architecture Container Image** — Automated publishing to GitHub Container Registry (GHCR).
- [ ] **Terminal Demo Recording** — ASCII/GIF demonstration of pruning and cache hits.
- [ ] **Distributed Cache Sync** — Multi-node synchronization using Redis/Valkey.

---

## Community & License

- **Discussions**: [GitHub Discussions](https://github.com/alexpatil/sukami/discussions)
- **Bug Reports**: [GitHub Issues](https://github.com/alexpatil/sukami/issues)
- **Maintainer**: Ganesh Shinde ([codingpenin@gmail.com](mailto:codingpenin@gmail.com))
- **License**: MIT License © Ganesh Shinde. See [LICENSE](LICENSE) for full terms.
