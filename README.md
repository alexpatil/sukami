<div align="center">

# ⚡ Sukami

**The High-Performance AI Token Optimization Gateway & Reverse Proxy**  
*Slash your OpenAI, Claude, and Groq bills by 30% to 100% with zero loss in output quality.*

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Zero External Dependencies](https://img.shields.io/badge/Dependencies-Standard_Lib_First-success.svg)]()
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()

[Features](#-key-features) •
[Real Benchmark](#-real-world-benchmark-results) •
[Quickstart](#-quickstart-in-30-seconds) •
[Architecture](#-architecture) •
[Documentation](docs/ARCHITECTURE.md)

</div>

---

## 💡 Why Sukami?

AI model providers charge **up to 5x more for output tokens than input tokens** (e.g. Claude 3.5 Sonnet: $3/M in vs. $15/M out). Moreover, models waste up to 40% of their output tokens generating conversational pleasantries, preambles, and repetitive disclaimers.

**Sukami solves this at the network layer.** By acting as a transparent, drop-in reverse proxy in front of your LLMs, Sukami:
1. **Prunes Input Prompts**: Strips redundant filler prefixes while strictly preserving code blocks and critical context.
2. **Compresses Output Tokens**: Transparently injects density directives to eliminate polite conversational fluff without losing factual or analytical depth.
3. **Caches Repetitive Queries**: Serves semantically equivalent queries from memory in **<10ms at $0.00 cost**.
4. **Governs Quotas & Budgets**: Prevents runaway loops with hard token caps and per-tenant rate limits.

---

## 📊 Real-World Benchmark Results

Tested on `openai/gpt-oss-120b` (Deep-dive enterprise comparative analysis between PostgreSQL and MongoDB for FinTech banking core):

| Metric | RAW (Direct Upstream) | Sukami Optimized | Direct Impact |
| :--- | :---: | :---: | :---: |
| **Output Tokens** | **3,072 tokens** | **1,979 tokens** | **1,093 tokens saved (35.58% cut)** |
| **Response Time (Latency)** | 7.33s | **4.50s** | **38.6% faster response** |
| **Completion Status** | `length` *(truncated mid-table)* | **`stop` *(100% complete)* | Full executive verdict delivered |
| **Cost Reduction** | Full rate billed | **35.5% savings** | **$753+ saved per 100k queries** |

> *Full benchmark logs and complete unedited comparison outputs are documented in [docs/BENCHMARKS.md](docs/BENCHMARKS.md).*

---

## ⚡ Quickstart in 30 Seconds

### 1. Clone & Run with Go
```bash
git clone https://github.com/your-username/sukami.git
cd sukami

# Configure environment
cp .env.example .env
# Add your GROQ_API_KEY or OPENAI_API_KEY to .env

# Start the Sukami Gateway & Dashboard
go run ./cmd/sukami
```
The Sukami Gateway and Web Dashboard will start immediately at `http://localhost:8080`.

---

## 🔌 1-Line Drop-in Integration

Sukami is **100% OpenAI-compatible**. Simply point your existing client's `base_url` to Sukami:

### Python
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",  # Point to Sukami
    api_key="sk-sukami-demo-12345"        # Or your generated Sukami secret key
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Explain distributed transaction logs"}]
)

print(response.choices[0].message.content)
```

### cURL
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-sukami-demo-12345" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Explain distributed transaction logs"}]
  }'
```

---

## 🏛 Architecture

Sukami is engineered with standard Go project layout principles:

```text
sukami/
├── cmd/
│   ├── sukami/              # Main gateway server entrypoint
│   └── sukami-bench/        # Standalone real-world benchmarking CLI
├── pkg/
│   ├── auth/                # API keys, rate limiting & free trial quotas
│   ├── cache/               # Thread-safe LRU + vector semantic caching
│   ├── config/              # Lightweight .env & configuration loader
│   ├── metrics/             # Real-time telemetry, token counts & cost tracking
│   ├── models/              # OpenAI schema structs
│   ├── optimizer/           # Prompt pruner & output density compressor
│   └── proxy/               # High-concurrency HTTP reverse proxy & SSE streaming
├── web/                     # React 19 + TypeScript modern SaaS dashboard
└── docs/                    # Architecture diagrams and benchmark reports
```

For complete technical specifications, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## 🧪 Testing & Verification

Sukami follows strict Go conventions and static type analysis:

```bash
# Run all Go unit tests
go test -v ./pkg/...

# Run Go static analysis
go vet ./...

# Run Frontend tests & linter
cd web && pnpm test && pnpm run lint
```

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
