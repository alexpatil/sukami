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
[Providers](#-supported-providers) •
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

Sukami is a **100% drop-in replacement** for the OpenAI API. Simply point your existing client's `base_url` to `http://localhost:8080/v1`:

### Python
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="your-api-key"
)

response = client.chat.completions.create(
    model="gpt-4o",  # or "llama-3.3-70b-versatile", "deepseek-chat"
    messages=[{"role": "user", "content": "Explain raft consensus in 3 bullet points"}]
)
print(response.choices[0].message.content)
```

### Node.js / TypeScript
```javascript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "http://localhost:8080/v1",
  apiKey: process.env.OPENAI_API_KEY,
});

const response = await client.chat.completions.create({
  model: "gpt-4o",
  messages: [{ role: "user", content: "Explain raft consensus in 3 bullet points" }],
});
```

### cURL
```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Explain raft consensus in 3 bullet points"}]
  }'
```

---

## 🌐 Supported Providers

Sukami automatically routes requests based on the requested model name with **zero configuration**:

| Provider | Models Routed Automatically |
| :--- | :--- |
| **OpenAI** | `gpt-4o`, `gpt-4o-mini`, `o1`, `gpt-4-turbo` |
| **Groq Cloud** | `llama-3.3-70b-versatile`, `mixtral-8x7b` |
| **DeepSeek** | `deepseek-chat` (V3), `deepseek-reasoner` (R1) |
| **OpenRouter** | `claude-3.5-sonnet`, `gemini-2.0`, and 300+ models |
| **Ollama / Local** | Local models via `http://localhost:11434/v1` |

> **Custom Routing**: Need a custom host? Pass `X-Upstream-Url: https://...` or `X-Provider: <id>` per request.

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
