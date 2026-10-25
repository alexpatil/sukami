<div align="center">

# ⚡ Sukami

**High-Performance AI Gateway & Token Optimizer in Pure Go**  
*Cut LLM costs and latency at the network layer with zero code changes.*

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Pure Go](https://img.shields.io/badge/Pure_Go-Static_Binary-success.svg)]()
[![Docker Image Size](https://img.shields.io/badge/Docker-~10MB-blue.svg)]()
[![Tests](https://img.shields.io/badge/Tests-Passing-brightgreen.svg)]()

[Features](#-key-features) •
[Quickstart](#-quickstart) •
[Integration](#-integration) •
[Providers & Routing](#-providers--routing) •
[Benchmark](#-measured-benchmark) •
[Contributing](CONTRIBUTING.md)

</div>

---

## 💡 Key Features

- **Transparent Prompt Pruning**: Strips redundant conversational fluff, pleasantries, and excessive whitespace using pure Go text optimization without altering prompt semantics or code integrity.
- **In-Memory Semantic Caching**: Serves semantically equivalent queries directly from cache in `<10ms` with an Entity Collision Guard (strictly distinguishes distinct entities like Docker vs. Kubernetes).
- **Tenant Cache Isolation**: Namespaces cache entries per API key to ensure zero cross-tenant data leaks.
- **Dynamic Multi-Provider Routing**: Switch providers dynamically per-request via headers (`X-Provider`, `X-Upstream-Url`) or model prefixes across OpenRouter, Groq, OpenAI, DeepSeek, and local Ollama/vLLM.
- **Real-Time Streaming**: Native Server-Sent Events (SSE) streaming with immediate chunk flushing.
- **Enterprise Concurrency**: Connection-pooled HTTP transport (1,000 idle conns), 10MB request body limits, Slowloris protection, and graceful shutdown.
- **Zero Dependencies**: 100% pure Go standard library binary (~10MB Alpine container, zero Node.js).

---

## ⚡ Quickstart

Get Sukami running locally in seconds:

### 1. Run with Go (Zero Dependencies)
```bash
git clone https://github.com/alexpatil/sukami.git
cd sukami
go run ./cmd/sukami
```

### 2. Run with Docker
```bash
git clone https://github.com/alexpatil/sukami.git
cd sukami
docker compose up -d
```

The gateway and developer dashboard will be immediately active at **`http://localhost:8080`**.

---

## 🔌 Integration

Sukami is a **100% drop-in replacement** for the OpenAI API. Simply set your client's `base_url` to `http://localhost:8080/v1`:

### Python
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-api-key",  # OpenAI, Groq, or OpenRouter API key
    default_headers={"X-Provider": "groq"}  # Optional: switch to groq, openrouter, deepseek, ollama
)

response = client.chat.completions.create(
    model="llama-3.3-70b-versatile",  # Or gpt-4o, claude-3.5-sonnet, etc.
    messages=[{"role": "user", "content": "Explain raft consensus in 3 bullet points"}]
)
print(response.choices[0].message.content)
```

### Node.js / TypeScript
```javascript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:8080/v1",
  apiKey: process.env.GROQ_API_KEY,
  defaultHeaders: { "X-Provider": "groq" }, // Optional: dynamic provider routing
});

const response = await client.chat.completions.create({
  model: "llama-3.3-70b-versatile",
  messages: [{ role: "user", content: "Explain raft consensus in 3 bullet points" }],
});
```

### cURL
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $GROQ_API_KEY" \
  -H "X-Provider: groq" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "llama-3.3-70b-versatile",
    "messages": [{"role": "user", "content": "Explain raft consensus in 3 bullet points"}]
  }'
```

---

## 🌐 Providers & Routing

Sukami forwards requests transparently to any OpenAI-compatible provider with no hardcoded model restrictions:

| Provider | Supported Models | Dynamic Routing Method |
| :--- | :--- | :--- |
| **OpenRouter** | 300+ models (Claude 3.5, Gemini 2.0, DeepSeek R1, Llama 3.3) | Header `X-Provider: openrouter` or `openrouter/<model>` |
| **Groq Cloud** | Ultra-fast LPU models (`llama-3.3-70b`, `mixtral-8x7b`) | Header `X-Provider: groq` |
| **OpenAI** | All official models (`gpt-4o`, `gpt-4o-mini`, `o1`, `gpt-4-turbo`) | Header `X-Provider: openai` (Default) |
| **DeepSeek** | `deepseek-chat` (V3), `deepseek-reasoner` (R1) | Header `X-Provider: deepseek` or `deepseek/<model>` |
| **Ollama / vLLM** | Local models (`llama3.2`, `mistral`, `deepseek-coder`) | Header `X-Provider: ollama` or `X-Upstream-Url: http://localhost:11434/v1` |
| **Custom Endpoints** | Any OpenAI-compatible host (Together, Fireworks, Mistral) | Header `X-Upstream-Url: https://...` |

---

## 📊 Measured Benchmark

Sample evaluation on `openai/gpt-oss-120b` (technical comparative analysis prompt):

| Metric | Direct Upstream | Via Sukami Proxy | Impact |
| :--- | :---: | :---: | :---: |
| **Output Tokens** | 3,072 tokens | **1,979 tokens** | **~35% fewer tokens** |
| **Response Time** | 7.33s | **4.50s** | **~38% lower latency** |
| **Completion Status** | `length` *(truncated)* | **`stop` *(complete)*** | Full response delivered |

> *Full benchmark methodology and automated runner are located in [cmd/sukami-bench](cmd/sukami-bench).*

---

## 🧪 Testing & Verification

Sukami enforces strict Go standards, race detection, and static type checking:

```bash
# Run unit tests across all packages
go test -v ./pkg/...

# Run Go static type analysis and vet checks
go vet ./...

# Build standalone production binary
go build -o bin/sukami ./cmd/sukami
```

---

## 🤝 Contributing

Contributions are welcome. Please review our [Contributing Guidelines](CONTRIBUTING.md) and [Code of Conduct](CODE_OF_CONDUCT.md) before submitting pull requests.

---

## 📄 License

This project is open-source under the [MIT License](LICENSE).
