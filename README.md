<div align="center">

# ⚡ Sukami

**High-Performance AI Token Optimization Gateway & Reverse Proxy**  
*Cut your LLM API expenses by 30% to 100% at the network layer with zero changes to your code.*

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Zero External Dependencies](https://img.shields.io/badge/Zero_Runtime_Bloat-Pure_Go-success.svg)]()
[![Tests](https://img.shields.io/badge/Tests-Passing-brightgreen.svg)]()
[![Status](https://img.shields.io/badge/Status-Alpha_v0.1.0-orange.svg)]()

[Why Sukami?](#-why-sukami) •
[Benchmark Results](#-real-world-benchmark-results) •
[Quickstart](#-quickstart-in-30-seconds) •
[Integration](#-1-line-drop-in-integration) •
[Architecture](#-architecture) •
[Technical Documentation](docs/ARCHITECTURE.md)

</div>

---

## 💡 Why Sukami?

AI model providers charge **up to 5x more for output tokens than input tokens** (e.g., Claude 3.5 Sonnet: $3/M in vs. $15/M out). Moreover, models waste up to 40% of their output tokens producing conversational fluff, pleasantries, preambles, and repetitive disclaimers.

**Sukami intercepts requests at the network layer as an ultra-fast reverse proxy:**

1. **Exact Byte-Pair Encoding (BPE)**: Integrated with OpenAI's official `tiktoken` BPE encodings (`cl100k_base`, `o200k_base`) with thread-safe in-memory caching.
2. **Output Token Compression**: Transparently guides output density to eliminate pleasantries and conversational padding without compromising analytical depth or code accuracy.
3. **5-Layer Semantic Cache**: Serves identical and semantically equivalent queries from memory in **<10ms at $0.00 cost**, protected by an **Entity Collision Guard** (e.g. strictly distinguishing Docker vs. Kubernetes).
4. **Flexible Dual-Key Resolution**: Pass your existing OpenAI/Groq key directly via standard `Authorization: Bearer sk-...` or `X-Upstream-Key`—no mandatory `.env` configuration required.
5. **100% Unlimited Open Source**: No arbitrary quota lockouts or artificial trial gates on self-hosted instances.

---

## 📊 Real-World Benchmark Results

Tested on `openai/gpt-oss-120b` (Deep-dive enterprise comparative analysis between PostgreSQL and MongoDB for a FinTech banking core):

| Metric | RAW (Direct Upstream) | Sukami Optimized | Direct Impact |
| :--- | :---: | :---: | :---: |
| **Output Tokens** | **3,072 tokens** | **1,979 tokens** | **1,093 tokens saved (35.58% cut)** |
| **Response Time (Latency)** | 7.33s | **4.50s** | **38.6% faster response** |
| **Completion Status** | `length` *(truncated mid-table)* | **`stop` *(100% complete)*** | Full executive verdict delivered |
| **Cost Reduction** | Full rate billed | **35.5% savings** | **$753+ saved per 100k queries** |

> *Full benchmark logs, methodology, and unedited outputs are documented in [docs/BENCHMARKS.md](docs/BENCHMARKS.md).*

---

## ⚡ Quickstart in 30 Seconds

Sukami is a pure Go binary with **zero Node.js dependencies** required for self-hosting.

### 1. Clone & Run
```bash
git clone https://github.com/alexpatil/sukami.git
cd sukami

# Start Sukami Gateway & Embedded Status Dashboard
go run ./cmd/sukami
```
The Sukami Gateway and Developer Status Dashboard will immediately launch at **`http://localhost:8080`**.

---

## 🔌 1-Line Drop-in Integration

Sukami is **100% OpenAI-compatible**. Simply point your existing client's `base_url` to Sukami:

### Python (OpenAI SDK)
```python
from openai import OpenAI

# Point to Sukami Gateway (Port 8080)
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="sk-proj-your-openai-or-groq-key"  # Direct passthrough or set in server .env
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Explain distributed systems architecture"}]
)

print(response.choices[0].message.content)
```

### Node.js (OpenAI SDK)
```javascript
import OpenAI from 'openai';

const client = new OpenAI({
  baseURL: 'http://localhost:8080/v1',
  apiKey: process.env.OPENAI_API_KEY || 'sk-proj-your-key',
});

const response = await client.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Explain distributed systems architecture' }],
});

console.log(response.choices[0].message.content);
```

### cURL
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-proj-your-key" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Explain distributed systems architecture"}]
  }'
```

---

## 🏛 Architecture

Sukami follows clean Go standard project layout conventions:

```text
sukami/
├── cmd/
│   ├── sukami/              # Main gateway server binary entrypoint
│   └── sukami-bench/        # Standalone real-world benchmarking CLI tool
├── pkg/
│   ├── auth/                # API keys and token-bucket client rate limiting
│   ├── cache/               # LRU cache, vector cosine matching & entity collision guards
│   ├── config/              # Lightweight environment configuration loader
│   ├── metrics/             # Real-time telemetry, token counts & cost tracking
│   ├── models/              # OpenAI schema structs & ChatML models
│   ├── optimizer/           # OpenAI tiktoken BPE tokenizer, prompt pruner & density engine
│   └── proxy/               # High-concurrency reverse proxy, dual-key auth & SSE streaming
├── web/                     # Lightweight zero-dependency developer status dashboard (HTML/CSS)
└── docs/                    # Architecture diagrams and benchmark reports
```

For complete technical specifications and sequence diagrams, see [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

---

## 🧪 Testing & Verification

Sukami enforces strict Go standards, race detection, and static type checking:

```bash
# Run all Go unit tests across all packages
go test -v ./...

# Run Go static type analysis and vet checks
go vet ./...

# Build standalone production binary
go build -o bin/sukami ./cmd/sukami
```

---

## 📄 License

This project is open-source under the [MIT License](LICENSE).
